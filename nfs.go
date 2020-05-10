package nfs

import (
	"bytes"
	"context"

	"github.com/go-git/go-billy/v5"
	"github.com/vmware/go-nfs-client/nfs/xdr"
)

const (
	nfsServiceID = 100003
)

func init() {
	RegisterMessageHandler(nfsServiceID, uint32(NFSProcedureNull), onNull)
	RegisterMessageHandler(nfsServiceID, uint32(NFSProcedureGetAttr), onGetAttr)
	RegisterMessageHandler(nfsServiceID, uint32(NFSProcedureFSInfo), onFSInfo)
}

func onNull(ctx context.Context, w *response, userHandle Handler) error {
	return w.Write([]byte{})
}

func onGetAttr(ctx context.Context, w *response, userHandle Handler) error {
	handle, err := xdr.ReadOpaque(w.req.Body)
	if err != nil {
		// TODO: wrap
		return err
	}

	fs, path, err := userHandle.FromHandle(handle)
	if err != nil {
		return &NFSStatusError{NFSStatusStale}
	}

	info, err := fs.Stat(path)
	if err != nil {
		// TODO: wrap
		return err
	}
	attr := ToFileAttribute(info)

	writer := bytes.NewBuffer([]byte{})
	if err := xdr.Write(writer, uint32(NFSStatusOk)); err != nil {
		return err
	}
	if err := xdr.Write(writer, attr); err != nil {
		return err
	}

	return w.Write(writer.Bytes())
}

const (
	// FSInfoPropertyLink does the FS support hard links?
	FSInfoPropertyLink = 0x0001
	// FSInfoPropertySymlink does the FS support soft links?
	FSInfoPropertySymlink = 0x0002
	// FSInfoPropertyHomogeneous does the FS need PATHCONF calls for each file
	FSInfoPropertyHomogeneous = 0x0008
	// FSInfoPropertyCanSetTime can the FS support setting access/mod times?
	FSInfoPropertyCanSetTime = 0x0010
)

func onFSInfo(ctx context.Context, w *response, userHandle Handler) error {
	roothandle, err := xdr.ReadOpaque(w.req.Body)
	if err != nil {
		// TODO: wrap
		return err
	}
	fs, path, err := userHandle.FromHandle(roothandle)
	if err != nil {
		// TODO: wrap
		return err
	}

	writer := bytes.NewBuffer([]byte{})
	if err := xdr.Write(writer, uint32(NFSStatusOk)); err != nil {
		return err
	}
	WritePostOpAttrs(writer, fs, path)

	type fsinfores struct {
		Rtmax       uint32
		Rtpref      uint32
		Rtmult      uint32
		Wtmax       uint32
		Wtpref      uint32
		Wtmult      uint32
		Dtpref      uint32
		Maxfilesize uint64
		TimeDelta   uint64
		Properties  uint32
	}

	res := fsinfores{
		Rtmax:       1 << 30,
		Rtpref:      1 << 30,
		Rtmult:      4096,
		Wtmax:       1 << 30,
		Wtpref:      1 << 30,
		Wtmult:      4096,
		Dtpref:      8192,
		Maxfilesize: 1 << 62, // wild guess. this seems big.
		TimeDelta:   1,       // nanosecond precision.
		Properties:  0,
	}

	// TODO: these aren't great indications of support, really.
	if _, ok := fs.(billy.Symlink); ok {
		res.Properties |= FSInfoPropertyLink
		res.Properties |= FSInfoPropertySymlink
	}
	// TODO: if the nfs share spans multiple virtual mounts, may need
	// to support granular PATHINFO responses.
	res.Properties |= FSInfoPropertyHomogeneous
	// TODO: not a perfect indicator
	if billy.CapabilityCheck(fs, billy.WriteCapability) {
		res.Properties |= FSInfoPropertyCanSetTime
	}
	// TODO: this whole struct should be specifiable by the userhandler.

	if err := xdr.Write(writer, res); err != nil {
		return err
	}
	return w.Write(writer.Bytes())
}