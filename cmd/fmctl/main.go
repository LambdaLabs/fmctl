package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/lambdalabs/fmctl/pkg/fmsdk"
)

var (
	address    = flag.String("address", "/var/run/nvidia-fabricmanager/nv-fabricmanager.sock", "FM daemon address or socket path")
	timeout    = flag.Uint("timeout", 5000, "Connection timeout in milliseconds")
	jsonOutput = flag.Bool("json", false, "Output in JSON format")
	verbose    = flag.Bool("verbose", false, "Verbose output")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <command> [arguments]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  list                    List all fabric partitions\n")
		fmt.Fprintf(os.Stderr, "  status <partition-id>   Show status of a specific partition\n")
		fmt.Fprintf(os.Stderr, "  activate <partition-id> Activate a fabric partition\n")
		fmt.Fprintf(os.Stderr, "  deactivate <partition-id> Deactivate a fabric partition\n")
		fmt.Fprintf(os.Stderr, "  info                    Show FM connection information\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	command := flag.Arg(0)

	// Handle info command without connection
	if command == "info" {
		cmdInfo()
		return
	}

	// Initialize FM library
	if ret := fmsdk.FMLibInit(); ret != fmsdk.FM_ST_SUCCESS {
		if *verbose {
			fmt.Fprintf(os.Stderr, "Warning: FMLibInit returned %v (library may not be available)\n", ret)
		}
	}
	defer fmsdk.FMLibShutdown()

	// Connect to FM daemon
	handle, ret := connectToFM()
	if ret != fmsdk.FM_ST_SUCCESS {
		fmt.Fprintf(os.Stderr, "Error: Failed to connect to Fabric Manager: %v\n", ret)
		os.Exit(1)
	}
	defer fmsdk.FMDisconnect(handle)

	// Execute command
	switch command {
	case "list":
		cmdList(handle)
	case "status":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "Error: status command requires partition-id argument\n")
			os.Exit(1)
		}
		partitionID, err := strconv.ParseUint(flag.Arg(1), 10, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid partition-id: %v\n", err)
			os.Exit(1)
		}
		cmdStatus(handle, uint32(partitionID))
	case "activate":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "Error: activate command requires partition-id argument\n")
			os.Exit(1)
		}
		partitionID, err := strconv.ParseUint(flag.Arg(1), 10, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid partition-id: %v\n", err)
			os.Exit(1)
		}
		cmdActivate(handle, uint32(partitionID))
	case "deactivate":
		if flag.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "Error: deactivate command requires partition-id argument\n")
			os.Exit(1)
		}
		partitionID, err := strconv.ParseUint(flag.Arg(1), 10, 32)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid partition-id: %v\n", err)
			os.Exit(1)
		}
		cmdDeactivate(handle, uint32(partitionID))
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n", command)
		flag.Usage()
		os.Exit(1)
	}
}

func connectToFM() (fmsdk.FMHandle, fmsdk.FMReturn) {
	params := fmsdk.FMConnectParams{
		Version:             1,
		AddressInfo:         *address,
		TimeoutMs:           uint32(*timeout),
		AddressIsUnixSocket: strings.HasSuffix(*address, ".sock"),
	}

	if *verbose {
		fmt.Printf("Connecting to Fabric Manager at %s (timeout: %dms, unix socket: %v)...\n",
			params.AddressInfo, params.TimeoutMs, params.AddressIsUnixSocket)
	}

	return fmsdk.FMConnect(params)
}

func cmdList(handle fmsdk.FMHandle) {
	partitions, ret := fmsdk.FMGetSupportedFabricPartitions(handle)
	if ret != fmsdk.FM_ST_SUCCESS {
		fmt.Fprintf(os.Stderr, "Error: Failed to get fabric partitions: %v\n", ret)
		os.Exit(1)
	}

	if *jsonOutput {
		data, err := json.MarshalIndent(partitions, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Table output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PARTITION ID\tSTATUS\tGPUs\tNVLINKS\tGPU PHYSICAL IDs")
	fmt.Fprintln(w, "------------\t------\t----\t-------\t----------------")

	for _, p := range partitions {
		status := "Inactive"
		if p.IsActive {
			status = "Active"
		}

		// Collect GPU physical IDs
		gpuIDs := make([]string, len(p.GPUInfo))
		totalNvLinks := uint32(0)
		for i, gpu := range p.GPUInfo {
			gpuIDs[i] = fmt.Sprintf("%d", gpu.PhysicalID)
			totalNvLinks += gpu.NumNvLinksAvailable
		}

		fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%s\n",
			p.PartitionID,
			status,
			p.NumGpus,
			totalNvLinks,
			strings.Join(gpuIDs, ","))
	}
	w.Flush()
}

func cmdStatus(handle fmsdk.FMHandle, partitionID uint32) {
	partitions, ret := fmsdk.FMGetSupportedFabricPartitions(handle)
	if ret != fmsdk.FM_ST_SUCCESS {
		fmt.Fprintf(os.Stderr, "Error: Failed to get fabric partitions: %v\n", ret)
		os.Exit(1)
	}

	var partition *fmsdk.FMPartitionInfo
	for i := range partitions {
		if partitions[i].PartitionID == partitionID {
			partition = &partitions[i]
			break
		}
	}
	if partition == nil {
		fmt.Fprintf(os.Stderr, "Error: Partition %d not found\n", partitionID)
		os.Exit(1)
	}

	if *jsonOutput {
		data, err := json.MarshalIndent(partition, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	// Detailed output
	fmt.Printf("Partition ID: %d\n", partition.PartitionID)
	fmt.Printf("Status: %s\n", map[bool]string{true: "Active", false: "Inactive"}[partition.IsActive])
	fmt.Printf("Number of GPUs: %d\n\n", partition.NumGpus)

	if len(partition.GPUInfo) > 0 {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "PHYSICAL ID\tUUID\tPCI BUS ID\tNVLINKS (Available/Max)\tRATE (MB/s)")
		fmt.Fprintln(w, "-----------\t----\t----------\t-----------------------\t-----------")

		for _, gpu := range partition.GPUInfo {
			uuid := gpu.UUID
			if uuid == "" {
				uuid = "N/A"
			}
			pciBusID := gpu.PCIBusID
			if pciBusID == "" {
				pciBusID = "N/A"
			}

			fmt.Fprintf(w, "%d\t%s\t%s\t%d/%d\t%d\n",
				gpu.PhysicalID,
				uuid,
				pciBusID,
				gpu.NumNvLinksAvailable,
				gpu.MaxNumNvLinks,
				gpu.NvlinkLineRateMBps)
		}
		w.Flush()
	}
}

func cmdActivate(handle fmsdk.FMHandle, partitionID uint32) {
	if *verbose {
		fmt.Printf("Activating partition %d...\n", partitionID)
	}

	ret := fmsdk.FMActivateFabricPartition(handle, partitionID)
	if ret != fmsdk.FM_ST_SUCCESS {
		fmt.Fprintf(os.Stderr, "Error: Failed to activate partition %d: %v\n", partitionID, ret)
		os.Exit(1)
	}

	if *jsonOutput {
		result := map[string]interface{}{
			"partitionId": partitionID,
			"action":      "activate",
			"status":      "success",
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Successfully activated partition %d\n", partitionID)
	}
}

func cmdDeactivate(handle fmsdk.FMHandle, partitionID uint32) {
	if *verbose {
		fmt.Printf("Deactivating partition %d...\n", partitionID)
	}

	ret := fmsdk.FMDeactivateFabricPartition(handle, partitionID)
	if ret != fmsdk.FM_ST_SUCCESS {
		fmt.Fprintf(os.Stderr, "Error: Failed to deactivate partition %d: %v\n", partitionID, ret)
		os.Exit(1)
	}

	if *jsonOutput {
		result := map[string]interface{}{
			"partitionId": partitionID,
			"action":      "deactivate",
			"status":      "success",
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("Successfully deactivated partition %d\n", partitionID)
	}
}

func cmdInfo() {
	info := map[string]interface{}{
		"address":        *address,
		"timeout":        *timeout,
		"isUnixSocket":   strings.HasSuffix(*address, ".sock"),
		"defaultAddress": "/var/run/nvidia-fabricmanager/nv-fabricmanager.sock",
		"defaultTimeout": 5000,
	}

	if *jsonOutput {
		data, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to marshal JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println("Fabric Manager Connection Info:")
		fmt.Printf("  Address: %s\n", info["address"])
		fmt.Printf("  Timeout: %dms\n", info["timeout"])
		fmt.Printf("  Unix Socket: %v\n", info["isUnixSocket"])
		fmt.Printf("\nDefaults:\n")
		fmt.Printf("  Default Address: %s\n", info["defaultAddress"])
		fmt.Printf("  Default Timeout: %dms\n", info["defaultTimeout"])
	}
}
