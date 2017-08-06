package utils

/***************************
* https://gitlab.weibo.cn/topweibo/idgen.git
*Genter a int64 Version 1
*|reserved 1bit|version 4bit|timestamp 32bit|instanceid 4bit|bid 4bit|increment 19bit|
*|reserved 1bit|version 4bit|timestamp 32bit|instanceid 4bit|bid 6bit|increment 17bit|
****************************/
import (
	"errors"
	"math/rand"
	"sync"
	"time"
	"strconv"
)

const (
	ARTICLE_BID_NORMAL = 5
	ARTICLE_BID_TOPARTICLE = 6
	ARTICLE_BID_LIVE = 7
	ARTICLE_BID_SLIDE = 8
	ARTICLE_BID_VIDEO = 9
	ARTICLE_BID_AD = 10
	ARTICLE_BID_WEIBO = 11
	ARTICLE_BID_CARD = 12
)

const (
	VERSION  = 2
	RAND_MAX = 0xff

	TIME_BITS    = 32
	VERSION_BITS = 4
	INSTID_BITS  = 4
	TIME_MASK    = (1 << TIME_BITS) - 1
	VERSION_MASK = (1 << VERSION_BITS) - 1
	INSTID_MASK  = (1 << INSTID_BITS) - 1

	NUM_BITS_V1 = 19
	BID_BITS_V1 = 4
	NUM_MASK_V1 = (1 << NUM_BITS_V1) - 1
	BID_MASK_V1 = (1 << BID_BITS_V1) - 1

	NUM_BITS_V2 = 17
	BID_BITS_V2 = 6
	NUM_MASK_V2 = (1 << NUM_BITS_V2) - 1
	BID_MASK_V2 = (1 << BID_BITS_V2) - 1
)

type IdGen struct {
	mutex sync.RWMutex
	temp  *ID
}

type ID struct {
	Version    int64 `json:"ver"`
	Time       int64 `json:"timestamp"`
	Instanceid int64 `json:"instanceid"`
	Num        int64 `json:"num"`
	Bid        int64 `json:"bid"`
}

func NewIdGen(i int8, ver int64) *IdGen {
	temp := &ID{
		Version:    ver,
		Instanceid: int64(i),
	}
	return &IdGen{
		temp: temp,
	}
}

func (self *IdGen) waitNextSecond() {
	nextSecond := time.Unix(self.temp.Time+1, 0)
	duration := nextSecond.Sub(time.Now())
	if duration <= 0 {
		return
	}
	time.Sleep(duration)
	return
}

func (self *IdGen) waitIfNeed(temp *ID) {
	var max int64 = NUM_MASK_V1
	if temp.Version == 2 {
		max = NUM_MASK_V2
	}
	if temp.Num >= max {
		self.waitNextSecond()
	}
}

func (self *IdGen) Gen(bid int64) (id int64) {
	self.mutex.Lock()
	defer self.mutex.Unlock()
	self.temp.Bid = bid
	self.temp.Num += 1
	self.waitIfNeed(self.temp)
	t := time.Now().Unix()
	if self.temp.Time != t {
		self.temp.Time = t
		self.temp.Num = rand.Int63n(RAND_MAX)
	}
	if self.temp.Version == 1 {
		id, _ = EncodeV1(self.temp)
	} else {
		id, _ = EncodeV2(self.temp)
	}

	return
}

func GetTimeFromId(id int64) (int64, error) {
	d, err := Decode(id)
	if err != nil {
		return 0, err
	}
	return d.Time, nil
}

func GetTimeFromIdString(idStr string) (int64, error) {
	id, err := strconv.ParseInt(idStr, 10, 0)
	if err != nil {
		WriteLog("debug", "GetTimeFromIdString failed:%s", idStr)
		return 0, err
	}
	return GetTimeFromId(id)
}
func GenMinIdByTime(t int64, ver int64) int64 {
	return ((ver & ((1 << VERSION_BITS) - 1)) << 59) | ((t << 27) & TIME_MASK)
}

func EncodeV2(i *ID) (int64, error) {
	if i.Version != 2 {
		return 0, errors.New("invalid version")
	}
	if i.Num > NUM_MASK_V2 {
		return 0, errors.New("num is too big")
	}
	if i.Bid > BID_MASK_V2 {
		return 0, errors.New("invalid bid")
	}
	if i.Instanceid > INSTID_MASK {
		return 0, errors.New("invalid instanceid")
	}
	return (i.Version << (63 - VERSION_BITS)) |
		(i.Time << (63 - VERSION_BITS - TIME_BITS)) |
		(i.Instanceid << (BID_BITS_V2 + NUM_BITS_V2)) |
		(i.Bid << NUM_BITS_V2) | (i.Num & NUM_MASK_V2), nil
}

func EncodeV1(i *ID) (int64, error) {
	if i.Version != 1 {
		return 0, errors.New("invalid version")
	}
	if i.Num > NUM_MASK_V1 {
		return 0, errors.New("num is too big")
	}
	if i.Bid > BID_MASK_V1 {
		return 0, errors.New("invalid bid")
	}
	if i.Instanceid > INSTID_MASK {
		return 0, errors.New("invalid instanceid")
	}
	return (i.Version << (63 - VERSION_BITS)) |
		(i.Time << (63 - VERSION_BITS - TIME_BITS)) |
		(i.Instanceid << (BID_BITS_V1 + NUM_BITS_V1)) |
		(i.Bid << NUM_BITS_V1) | (i.Num & NUM_MASK_V1), nil
}

func Decode(id int64) (*ID, error) {
	ver := id >> (63 - VERSION_BITS)
	switch ver {
	case 1:
		return DecodeV1(id)
	case 2:
		return DecodeV2(id)
	}
	return nil, errors.New("invalid id")
}

func DecodeV1(id int64) (*ID, error) {
	if id>>(63-VERSION_BITS) != 1 {
		return nil, errors.New("invalid ID")
	}
	d := new(ID)
	d.Version = 1
	d.Time = (id >> (NUM_BITS_V1 + BID_BITS_V1 + INSTID_BITS)) & TIME_MASK
	d.Bid = (id >> NUM_BITS_V1) & BID_MASK_V1
	d.Instanceid = (id >> (BID_BITS_V1 + NUM_BITS_V1)) & INSTID_MASK
	d.Num = id & NUM_MASK_V1
	return d, nil
}

func DecodeV2(id int64) (*ID, error) {
	if id>>(63-VERSION_BITS) != 2 {
		return nil, errors.New("invalid ID")
	}
	d := new(ID)
	d.Version = 2
	d.Time = (id >> (NUM_BITS_V2 + BID_BITS_V2 + INSTID_BITS)) & TIME_MASK
	d.Bid = (id >> NUM_BITS_V2) & BID_MASK_V2
	d.Instanceid = (id >> (BID_BITS_V2 + NUM_BITS_V2)) & INSTID_MASK
	d.Num = id & NUM_MASK_V2
	return d, nil
}
