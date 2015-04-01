package llapi

//
// #cgo LDFLAGS: -llustreapi
// #include <fcntl.h>
// #include <stdlib.h>
// #include <lustre/lustreapi.h>
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"
)

// HsmUserAction specifies an action for HsmRequest().
type HsmUserAction uint

const (
	UserNone    = HsmUserAction(C.HUA_NONE)
	UserArchive = HsmUserAction(C.HUA_ARCHIVE)
	UserRestore = HsmUserAction(C.HUA_RESTORE)
	UserRelease = HsmUserAction(C.HUA_RELEASE)
	UserRemove  = HsmUserAction(C.HUA_REMOVE)
	UserCancel  = HsmUserAction(C.HUA_CANCEL)
)

func (action HsmUserAction) String() string {
	return C.GoString(C.hsm_user_action2name(C.enum_hsm_user_action(action)))
}

// HsmRequest submits an HSM request for list of files
// The max suported size of the fileList is about 50.
func HsmRequest(r string, cmd HsmUserAction, archiveID uint, fids []*CFid) (int, error) {
	fileCount := len(fids)
	if fileCount < 1 {
		return 0, fmt.Errorf("Request must include at least 1 file!")
	}

	hur := C.llapi_hsm_user_request_alloc(C.int(fileCount), 0)
	defer C.free(unsafe.Pointer(hur))
	if hur == nil {
		panic("Failed to allocate HSM User Request struct!")
	}

	hur.hur_request.hr_action = C.__u32(cmd)
	hur.hur_request.hr_archive_id = C.__u32(archiveID)
	hur.hur_request.hr_flags = 0
	hur.hur_request.hr_itemcount = 0
	hur.hur_request.hr_data_len = 0

	// https://code.google.com/p/go-wiki/wiki/cgo#Turning_C_arrays_into_Go_slices
	hdr := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&hur.hur_user_item)),
		Len:  fileCount,
		Cap:  fileCount,
	}
	userItems := *(*[]C.struct_hsm_user_item)(unsafe.Pointer(&hdr))
	for i, f := range fids {
		userItems[i].hui_extent.offset = 0
		userItems[i].hui_extent.length = C.__u64(^uint(0))
		userItems[i].hui_fid = (C.struct_lu_fid)(*f)
		hur.hur_request.hr_itemcount++
	}

	num := int(hur.hur_request.hr_itemcount)
	if num != fileCount {
		return 0, fmt.Errorf("lustre: Can't submit incomplete request (%d/%d)", num, fileCount)
	}

	buf := C.CString(r)
	defer C.free(unsafe.Pointer(buf))
	rc, err := C.llapi_hsm_request(buf, hur)
	if err := isError(rc, err); err != nil {
		return 0, fmt.Errorf("lustre: Got error from llapi_hsm_request: %s", err.Error())
	}

	return num, nil
}
