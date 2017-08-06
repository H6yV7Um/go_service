package memcache

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"unsafe"
)

const (
	DEFAULT_COMPRESS_LEVEL = 7
)

func zlib4php(content []byte) (result []byte, err error) {
	var b bytes.Buffer
	tmp := content
	compresser, err := zlib.NewWriterLevel(&b, DEFAULT_COMPRESS_LEVEL)
	if err != nil {
		return
	}
	compresser.Write(content)
	compresser.Close()
	tmp = b.Bytes()
	tmpLen := uint32(len(content))
	binLen := make([]byte, unsafe.Sizeof(tmpLen))
	binary.LittleEndian.PutUint32(binLen, tmpLen)
	result = bytes.Join([][]byte{binLen, tmp}, []byte{})
	return
}

func dezlib4php(content []byte) (result []byte, err error) {
	var orilen uint32
	lenbuf := content[:unsafe.Sizeof(orilen)]
	orilen = binary.LittleEndian.Uint32(lenbuf)
	buffer := content[unsafe.Sizeof(orilen):]
	b := bytes.NewReader(buffer)
	decompresser, err := zlib.NewReader(b)
	if err != nil {
		return
	}
	var r bytes.Buffer
	rw := bufio.NewWriter(&r)
	io.Copy(rw, decompresser)
	result = r.Bytes()
	if len(result) != int(orilen) {
		err = errors.New("decompress content length not equl origin length")
	}
	return
}
