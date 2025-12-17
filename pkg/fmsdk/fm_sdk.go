package fmsdk

/*
#cgo !dev LDFLAGS: -lnvfm
#include "nv_fm_agent.h"
*/

import "C"
import (
	"fmt"
	"unsafe"
)

// Go wrapper types for FM SDK
type FMReturn int
type FMHandle unsafe.Pointer

// FM SDK return codes - matching NVIDIA SDK exactly
const (
	FM_ST_SUCCESS              FMReturn = C.FM_ST_SUCCESS
	FM_ST_BADPARAM             FMReturn = C.FM_ST_BADPARAM
	FM_ST_GENERIC_ERROR        FMReturn = C.FM_ST_GENERIC_ERROR
	FM_ST_NOT_SUPPORTED        FMReturn = C.FM_ST_NOT_SUPPORTED
	FM_ST_UNINITIALIZED        FMReturn = C.FM_ST_UNINITIALIZED
	FM_ST_TIMEOUT              FMReturn = C.FM_ST_TIMEOUT
	FM_ST_VERSION_MISMATCH     FMReturn = C.FM_ST_VERSION_MISMATCH
	FM_ST_IN_USE               FMReturn = C.FM_ST_IN_USE
	FM_ST_NOT_CONFIGURED       FMReturn = C.FM_ST_NOT_CONFIGURED
	FM_ST_CONNECTION_NOT_VALID FMReturn = C.FM_ST_CONNECTION_NOT_VALID
	FM_ST_NVLINK_ERROR         FMReturn = C.FM_ST_NVLINK_ERROR
)

// FMConnectParams represents connection parameters for FM SDK
type FMConnectParams struct {
	Version             uint32
	AddressInfo         string
	TimeoutMs           uint32
	AddressIsUnixSocket bool
}

// FMFabricPartitionGpuInfo represents GPU information in a fabric partition
type FMFabricPartitionGpuInfo struct {
	PhysicalID          uint32
	UUID                string
	PCIBusID            string
	NumNvLinksAvailable uint32
	MaxNumNvLinks       uint32
	NvlinkLineRateMBps  uint32
}

// String representation of FM return codes
func (r FMReturn) String() string {
	switch r {
	case FM_ST_SUCCESS:
		return "Success"
	case FM_ST_BADPARAM:
		return "Bad parameter"
	case FM_ST_GENERIC_ERROR:
		return "Generic error"
	case FM_ST_NOT_SUPPORTED:
		return "Not supported"
	case FM_ST_UNINITIALIZED:
		return "Uninitialized"
	case FM_ST_TIMEOUT:
		return "Timeout"
	case FM_ST_VERSION_MISMATCH:
		return "Version mismatch"
	case FM_ST_IN_USE:
		return "In use"
	case FM_ST_NOT_CONFIGURED:
		return "Not configured"
	case FM_ST_CONNECTION_NOT_VALID:
		return "Connection not valid"
	case FM_ST_NVLINK_ERROR:
		return "NVLink error"
	default:
		return fmt.Sprintf("Unknown error (%d)", int(r))
	}
}

// Error implementation
func (r FMReturn) Error() string {
	return r.String()
}

// FMPartitionInfo represents a fabric partition configuration
type FMPartitionInfo struct {
	PartitionID uint32
	IsActive    bool
	NumGpus     uint32
	GPUInfo     []FMFabricPartitionGpuInfo
}

// Go wrapper functions for FM SDK

// FMLibInit initializes the Fabric Manager library
func FMLibInit() FMReturn {
	return FMReturn(C.fmLibInit())
}

// FMLibShutdown shuts down the Fabric Manager library
func FMLibShutdown() FMReturn {
	return FMReturn(C.fmLibShutdown())
}

// FMConnect connects to a running Fabric Manager instance
func FMConnect(params FMConnectParams) (FMHandle, FMReturn) {
	// Convert Go struct to C struct
	var cParams C.fmConnectParams_v1
	// Use the NVIDIA SDK version macro
	cParams.version = C.fmConnectParams_version
	cParams.timeoutMs = C.uint(params.TimeoutMs)

	// Set address info and socket flag
	cAddr := C.CString(params.AddressInfo)
	defer C.free(unsafe.Pointer(cAddr))
	copy((*[256]C.char)(unsafe.Pointer(&cParams.addressInfo[0]))[:], (*[256]C.char)(unsafe.Pointer(cAddr))[:])

	if params.AddressIsUnixSocket {
		cParams.addressIsUnixSocket = 1
	} else {
		cParams.addressIsUnixSocket = 0
	}

	var handle C.fmHandle_t
	ret := FMReturn(C.fmConnect(&cParams, &handle))
	return FMHandle(handle), ret
}

// FMDisconnect disconnects from Fabric Manager instance
func FMDisconnect(handle FMHandle) FMReturn {
	return FMReturn(C.fmDisconnect(C.fmHandle_t(handle)))
}

// FMGetSupportedFabricPartitions queries supported fabric partitions
func FMGetSupportedFabricPartitions(handle FMHandle) ([]FMPartitionInfo, FMReturn) {
	// Allocate C struct for partition list (corrected to match NVIDIA API)
	var partitionList C.fmFabricPartitionList_t
	// Use the NVIDIA SDK version macro
	partitionList.version = C.fmFabricPartitionList_version

	ret := FMReturn(C.fmGetSupportedFabricPartitions(
		C.fmHandle_t(handle),
		&partitionList,
	))

	if ret != FM_ST_SUCCESS {
		return nil, ret
	}

	// Convert C structs to Go structs
	numPartitions := int(partitionList.numPartitions)
	result := make([]FMPartitionInfo, numPartitions)

	for i := 0; i < numPartitions; i++ {
		partition := partitionList.partitionInfo[i]

		// Convert GPU info array
		gpuInfo := make([]FMFabricPartitionGpuInfo, partition.numGpus)
		for j := 0; j < int(partition.numGpus); j++ {
			gpu := partition.gpuInfo[j]

			// Note: On DGX H100/HGX H100+ systems, UUID and PCI Bus ID may be empty
			// GPU Physical ID should be used for correlation with nvidia-smi GPU Module ID
			uuid := C.GoString(&gpu.uuid[0])
			pciBusID := C.GoString(&gpu.pciBusId[0])

			gpuInfo[j] = FMFabricPartitionGpuInfo{
				PhysicalID:          uint32(gpu.physicalId),
				UUID:                uuid,
				PCIBusID:            pciBusID,
				NumNvLinksAvailable: uint32(gpu.numNvLinksAvailable),
				MaxNumNvLinks:       uint32(gpu.maxNumNvLinks),
				NvlinkLineRateMBps:  uint32(gpu.nvlinkLineRateMBps),
			}
		}

		result[i] = FMPartitionInfo{
			PartitionID: uint32(partition.partitionId),
			IsActive:    partition.isActive != 0,
			NumGpus:     uint32(partition.numGpus),
			GPUInfo:     gpuInfo,
		}
	}

	return result, ret
}

// FMActivateFabricPartition activates a fabric partition
func FMActivateFabricPartition(fmHandle FMHandle, partitionID uint32) FMReturn {
	ret := FMReturn(C.fmActivateFabricPartition(
		C.fmHandle_t(fmHandle),
		C.fmFabricPartitionId_t(partitionID),
	))
	return ret
}

// FMDeactivateFabricPartition deactivates a fabric partition
func FMDeactivateFabricPartition(fmHandle FMHandle, partitionID uint32) FMReturn {
	return FMReturn(C.fmDeactivateFabricPartition(
		C.fmHandle_t(fmHandle),
		C.fmFabricPartitionId_t(partitionID),
	))
}
