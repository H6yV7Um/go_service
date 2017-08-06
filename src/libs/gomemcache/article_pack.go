package memcache

import (
	"bytes"
	"io"
	"bufio"
	"errors"
	"reflect"
)

const (
	articleModelVersion = 20161108
)

type ArticleEncoder struct {
	w 	 *bytes.Buffer
	buf  []byte
	err  error
}

func NewArticleEncoder(bw *bytes.Buffer) *ArticleEncoder {
	return &ArticleEncoder{
		w:    bw,
		buf:  make([]byte, 9),
	}
}

func (e *ArticleEncoder) write(b []byte) {
	if e.err != nil {
		return
	}
	n, err := e.w.Write(b)
	if err != nil {
		e.err = err
		return
	}
	if n < len(b) {
		e.err = io.ErrShortWrite
		return
	}
	return
}

func (e *ArticleEncoder) WriteInt32(n uint32) {
	if e.err != nil {
		return
	}
	e.buf = e.buf[:4]
	e.buf[0] = byte(n >> 24)
	e.buf[1] = byte(n >> 16)
	e.buf[2] = byte(n >> 8)
	e.buf[3] = byte(n)
	e.write(e.buf)
}

func (e *ArticleEncoder) WriteInt64(n uint64) {
	if e.err != nil {
		return
	}
	e.buf = e.buf[:8]
	e.buf[0] = byte(n >> 56)
	e.buf[1] = byte(n >> 48)
	e.buf[2] = byte(n >> 40)
	e.buf[3] = byte(n >> 32)
	e.buf[4] = byte(n >> 24)
	e.buf[5] = byte(n >> 16)
	e.buf[6] = byte(n >> 8)
	e.buf[7] = byte(n)
	e.write(e.buf)
}

func (e *ArticleEncoder) WriteInt(n int) {
	e.WriteInt64(uint64(n))
}

func (e *ArticleEncoder) WriteString(s string) {
	if e.err != nil {
		return
	}
	e.WriteInt32(uint32(len(s)))
	if e.err != nil {
		return
	}
	e.w.Write([]byte(s))
}

func (e *ArticleEncoder) WriteInterface(i interface{}, t reflect.Kind) {
	switch t {
	case reflect.Int:
		n, ok := i.(int)
		if !ok {
			e.err = errors.New("not a int")
		} else {
			e.WriteInt(n)
		}
	case reflect.Int64:
		n, ok := i.(int64)
		if !ok {
			e.err = errors.New("not a int64")
		} else {
			e.WriteInt64(uint64(n))
		}
	case reflect.String:
		s, ok := i.(string)
		if !ok {
			e.err = errors.New("not a string")
		} else {
			e.WriteString(s)
		}
	default:
		e.err = errors.New("type not support")
	}
}

func MarshalArticle(article map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	e := NewArticleEncoder(&buf)
	e.WriteInt32(articleModelVersion)
	e.WriteInterface(article["id"], reflect.String)
	e.WriteInterface(article["uid"], reflect.Int64)
	e.WriteInterface(article["oid"], reflect.String)
	e.WriteInterface(article["time"], reflect.Int64)
	e.WriteInterface(article["updtime"], reflect.Int64)
	e.WriteInterface(article["type"], reflect.Int)
	e.WriteInterface(article["category"], reflect.String)
	e.WriteInterface(article["source"], reflect.String)
	e.WriteInterface(article["title"], reflect.String)
	e.WriteInterface(article["url"], reflect.String)
	e.WriteInterface(article["original_url"], reflect.String)
	e.WriteInterface(article["tags"], reflect.String)
	e.WriteInterface(article["images"], reflect.String)
	e.WriteInterface(article["content"], reflect.String)
	e.WriteInterface(article["flag"], reflect.Int)
	e.WriteInterface(article["mid"], reflect.String)
	e.WriteInterface(article["cover"], reflect.String)
	e.WriteInterface(article["created_time"], reflect.Int64)
	e.WriteInterface(article["abstract"], reflect.String)
	e.WriteInterface(article["cms_options"], reflect.String)
	e.WriteInterface(article["gid"], reflect.Int64)
	e.WriteInterface(article["display_id"], reflect.Int64)
	e.WriteInterface(article["top_groupid"], reflect.String)
	e.WriteInterface(article["article_type"], reflect.String)
	
	return buf.Bytes(), e.err
}

type ArticleDecoder struct {
	r   *bufio.Reader
	buf []byte
	err error
}

func NewArticleDecoder(b []byte) *ArticleDecoder {
	return &ArticleDecoder{
		r:    bufio.NewReader(bytes.NewReader(b)),
		buf:  make([]byte, 64),
	}
}

func (d *ArticleDecoder) ReadInt32() int32 {
	if d.err != nil {
		return 0
	}
	b := d.buf[:4]
	_, d.err = io.ReadFull(d.r, b)
	if d.err != nil {
		return 0
	}
	n := (uint32(b[0]) << 24) |
		(uint32(b[1]) << 16) |
		(uint32(b[2]) << 8) |
		uint32(b[3])
	
	return int32(n)
}

func (d *ArticleDecoder) ReadInt64() int64 {
	if d.err != nil {
		return 0
	}
	b := d.buf[:8]
	_, d.err = io.ReadFull(d.r, b)
	if d.err != nil {
		return 0
	}
	n := (uint64(b[0]) << 56) |
		(uint64(b[1]) << 48) |
		(uint64(b[2]) << 40) |
		(uint64(b[3]) << 32) |
		(uint64(b[4]) << 24) |
		(uint64(b[5]) << 16) |
		(uint64(b[6]) << 8) |
		uint64(b[7])
	
	return int64(n)
}

func (d *ArticleDecoder) ReadInt() int {
	return int(d.ReadInt64())
}

func (d *ArticleDecoder) ReadString() string {
	if d.err != nil {
		return ""
	}
	n := d.ReadInt32()
	if d.err != nil {
		return ""
	}
	b := make([]byte, n)
	_, d.err = io.ReadFull(d.r, b)
	if d.err != nil {
		return ""
	}
	return string(b)
}

func UnmarshalArticle(b []byte) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	d := NewArticleDecoder(b)
	version := d.ReadInt32()
	if version != articleModelVersion {
		return result, errors.New("version not match")
	}
	result["id"] = d.ReadString()
	result["uid"] = d.ReadInt64()
	result["oid"] = d.ReadString()
	result["time"] = d.ReadInt64()
	result["updtime"] = d.ReadInt64()
	result["type"] = d.ReadInt()
	result["category"] = d.ReadString()
	result["source"] = d.ReadString()
	result["title"] = d.ReadString()
	result["url"] = d.ReadString()
	result["original_url"] = d.ReadString()
	result["tags"] = d.ReadString()
	result["images"] = d.ReadString()
	result["content"] = d.ReadString()
	result["flag"] = d.ReadInt()
	result["mid"] = d.ReadString()
	result["cover"] = d.ReadString()
	result["created_time"] = d.ReadInt64()
	result["abstract"] = d.ReadString()
	result["cms_options"] = d.ReadString()
	result["gid"] = d.ReadInt64()
	result["display_id"] = d.ReadInt64()
	result["top_groupid"] = d.ReadString()
	result["article_type"] = d.ReadString()
	
	return result, d.err
}
