package libio

import (
	"encoding/binary"
	"io"
)

type BinaryReader struct {
	Order     binary.ByteOrder
	Src       io.Reader
	Index     int
	LastIndex int
	Err       error
	buf       []byte
}

func (br *BinaryReader) ReadBytes(n int) (ok bool) {
	if br.Err != nil {
		return false
	}

	if cap(br.buf) <= n {
		br.buf = make([]byte, n)
	} else {
		br.buf = br.buf[:n]
	}

	nread, err := br.Src.Read(br.buf)
	if err != nil {
		br.Err = err
		ok = false
	}

	br.LastIndex = br.Index
	br.Index += nread

	return br.Err == nil
}

func (br *BinaryReader) Read(p []byte) (n int, err error) {
	return br.Src.Read(p)
}

func (br *BinaryReader) ReadUInt8(i *int) (ok bool) {
	if !br.ReadBytes(1) {
		return false
	}
	*i = int(br.buf[0])
	return true
}

func (br *BinaryReader) ReadUInt16(i *int) (ok bool) {
	if !br.ReadBytes(2) {
		return false
	}
	*i = int(br.Order.Uint16(br.buf))
	return true
}

func (br *BinaryReader) ReadUInt32(i *int) (ok bool) {
	if !br.ReadBytes(4) {
		return false
	}
	*i = int(br.Order.Uint32(br.buf))
	return true
}

func (br *BinaryReader) ReadRef(data any) (ok bool) {
	if br.Err != nil {
		return false
	}
	err := binary.Read(br.Src, br.Order, data)
	br.Err = err
	br.LastIndex = br.Index
	if err == nil {
		br.Index += binary.Size(data)
	}
	return err == nil
}

type BinaryWriter struct {
	Order binary.ByteOrder
	Dst   io.Writer
	Err   error
}

func (bw *BinaryWriter) WriteBytes(p []byte) (ok bool) {
	if bw.Err != nil {
		return false
	}

	_, err := bw.Dst.Write(p)
	if err != nil {
		bw.Err = err
		return false
	}
	return true
}

func (bw *BinaryWriter) Write(p []byte) (n int, err error) {
	return bw.Dst.Write(p)
}

func (bw *BinaryWriter) WriteUInt32(i uint32) (ok bool) {
	buf := make([]byte, 4)
	bw.Order.PutUint32(buf, i)
	return bw.WriteBytes(buf)
}

func (bw *BinaryWriter) WriteUInt16(i uint16) (ok bool) {
	buf := make([]byte, 2)
	bw.Order.PutUint16(buf, i)
	return bw.WriteBytes(buf)
}

func (bw *BinaryWriter) WriteRef(data any) (ok bool) {
	if bw.Err != nil {
		return false
	}
	err := binary.Write(bw.Dst, bw.Order, data)
	bw.Err = err
	return err == nil
}
