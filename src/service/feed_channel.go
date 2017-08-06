package service

import (
	"errors"
	"strconv"
	"fmt"
	"math"
	"strings"
	"hash/crc32"
	"utils"
	"math/rand"
	"io/ioutil"
	"encoding/json"
)

const (
	rChanFeedManualPrefix = "feed_suggest_"
	rArticleInfoPrefix = "N:"
	rChanHisPrefix = "IH:"
	rCardPrefix = "string:card:"
	artFieldNewID = "id"
	artFieldHasPic = "im"
	artFieldType = "ty"
	version_cur = "245"
	F_VERSION int = 0x1 //按二进制位来做标识
	F_HAS_PIC int = 0x2
	F_SET_TOP int = 0x4
	BID_CARD int64 = 12
	CHAN_HIS_RD_R int = 0
	CHAN_HIS_RD_W int = 1
	BID_CHANNEL int64 = 15
)

type ChanFeedGetArgs struct {
	CommonArgs
	ChanID    string `msgpack:"chanid" must:"1" comment:"uid"`
	SubChanID string `msgpack:"subchanid" must:"1" comment:"uid"`
	Uid       int64 `msgpack:"uid" must:"1" comment:"uid"`
	Imei      string `msgpack:"imei" must:"1" comment:"uid"`
	Num       int `msgpack:"num" must:"1" comment:"uid"`
	Version   string `msgpack:"version" must:"1" comment:"uid"`
}

type ChanFeedGetResult struct {
	Result Result
	Data   *ChanFeedGetData
}

type ChanFeedGetData struct {
	ArticleIDs []string
	Bid        int64
}

func (f *Feed)GetChanFeed(args *ChanFeedGetArgs, reply *ChanFeedGetResult) error {
	//validate param
	if err := f.validateChanParam(args); err != nil {
		reply.Result = Result{Status:1, Error:err.Error()}
		return err
	}

	//get history
	history, err := f.getChanHistory(args)
	if err != nil {
		reply.Result = Result{Status:1, Error:err.Error()}
		return err
	}

	//get manual data
	idsManual, err := f.getChanManualData(args, history)
	if err != nil {
		reply.Result = Result{Status:1, Error:err.Error()}
		return err
	}

	reply.Data = new(ChanFeedGetData)

	numMachine := 0
	if len(idsManual) < args.Num {
		numMachine = args.Num - len(idsManual)
		//get machine data
		idsMachine, err := f.getChanMachineData(args, numMachine, history)
		if err != nil {
			reply.Result = Result{Status:1, Error:err.Error()}
			return err
		}

		fmt.Println(idsManual)
		fmt.Println(idsMachine)
		fmt.Println(reply.Data)
		//return nil

		reply.Data.ArticleIDs = append(idsManual, idsMachine...)
	} else {
		reply.Data.ArticleIDs = idsManual[:args.Num]
	}

	reply.Data.Bid = f.genID(BID_CHANNEL)
	reply.Result = Result{Status:0, Error:"success"}
	return nil
}

func (f *Feed)validateChanParam(args *ChanFeedGetArgs) error {
	if args.ChanID == "" {
		f.Error("chanParamValidate error:ChanID is empty")
		return errors.New("ChanID is empty")
	}
	if args.Num <= 0 {
		f.Error("chanParamValidate error:Num is invalid")
		return errors.New("Num is invalid")
	}
	if args.Uid == 0&&args.Imei == "" {
		f.Error("chanParamValidate error:uid and imei is empty")
		return errors.New("uid and imei is empty")
	}
	return nil
}

func (f *Feed)getChanHistory(args *ChanFeedGetArgs) (history map[string]int, err error) {
	rChanHisR, rChanHisKey, err := f.getChanHisRedis(args, CHAN_HIS_RD_R)
	if err != nil {
		f.Error("getChanHistory redis error:%s", err.Error())
		err = errors.New("getChanHistory redis error")
		return
	}
	historyTmp, err := rChanHisR.LRange(rChanHisKey, 0, f.Ctx.Cfg.Feed.ChanFeedHistoryNum - 1)
	if err != nil {
		f.Error("getChanHistory error:%s", err.Error())
		err = errors.New("getChanHistory error")
		return
	}
	history = make(map[string]int)
	for k := range historyTmp {
		history[historyTmp[k]] = 1
	}
	return
}

func (f *Feed)getChanManualData(args *ChanFeedGetArgs, history map[string]int) (ids []string, err error) {
	ids = make([]string, 0)
	//视频子频道不存在人工数据,热点频道不读取人工数据
	if (args.ChanID == "10015"&&args.SubChanID != "") || args.ChanID == "1" {
		return
	}
	r, err := f.Ctx.Redis("star_feed_r", 0)
	if err != nil {
		f.Error("get star_feed_r redis error:%s", err.Error())
		err = errors.New("get star_feed_r redis error")
		return
	}

	feedKey := rChanFeedManualPrefix + args.ChanID
	manualAll, err := r.ZRevRangeWithScore(feedKey, 0, -1)
	if err != nil {
		f.Error("get channel feed manual data error:%s", err.Error())
		err = errors.New("get channel feed manual data error")
		return
	}
	mannulNum := f.Ctx.Cfg.Feed.ChanFeedManualNum
	ids, err = f.filter(manualAll, mannulNum, F_VERSION | F_SET_TOP, history, args)
	if err != nil {
		err = errors.New(fmt.Sprintf("getChanManualData error:%s", err.Error()))
	}
	return
}

func (f *Feed)getChanMachineData(args *ChanFeedGetArgs, num int, history map[string]int) (ids []string, err error) {
	r, err := f.Ctx.Redis("star_feed_r", 0)
	if err != nil {
		f.Error("get star_feed_r redis error:%s", err.Error())
		err = errors.New("get star_feed_r redis error")
		return
	}

	feedKey := getChanMachineKey(args.ChanID, args.SubChanID)
	machineNum := f.Ctx.Cfg.Feed.ChanFeedMachineNum
	machineAll, err := r.ZRevRangeWithScore(feedKey, 0, machineNum - 1)
	if err != nil {
		f.Error("get channel feed machine data error:%s", err.Error())
		err = errors.New("get channel feed machine data error")
		return
	}
	ids, err = f.filter(machineAll, num, F_VERSION | F_HAS_PIC, history, args)
	if err != nil {
		err = errors.New(fmt.Sprintf("getChanMachineData error:%s", err.Error()))
	}
	return
}

func (f *Feed)filter(input []string, num int, flag int, history map[string]int, args *ChanFeedGetArgs) (output []string, err error) {
	//为了兼容置顶文章的score判断,这里的input要求必须带score
	if input == nil || len(input) % 2 != 0 || num < 0 {
		f.Error("filter error:invalid params")
		err = errors.New("filter error:invalid params")
		return
	}

	output = make([]string, 0)
	if num == 0 {
		return
	}

	outputHasPic := make([]string, 0)
	outputNorPic := make([]string, 0)
	historyTmp := make(map[string]map[string]string)

	countTotal := 0 //过滤有图无图——总计数
	countHasPic := 0 //过滤有图无图——有图计数
	numHasPic := int(math.Ceil(float64(num) / 2)) //过滤有图无图——有图数

	for i := 0; i < len(input); i += 2 {
		oid := input[i]
		//组card标识
		cardFlag := false
		if !strings.Contains(oid, ":") {
			idInt64, err := strconv.ParseInt(oid, 10, 64)
			if err != nil {
				f.Error("Parse oid error:%s,oid:%s", err.Error(), oid)
				continue
			}
			m := f.decID(idInt64)
			bidI, ok := m["bid"]
			if !ok {
				f.Error("decID error:no bid")
				continue
			}
			bidj, ok := bidI.(json.Number)
			if !ok {
				f.Error("decID error:bid is not json.Number")
				continue
			}
			bid, _ := bidj.Int64()
			if bid != BID_CARD {
				continue
			}
			cardFlag = true
		} else {
			artInfo, err := f.getArticleInfo(oid)
			if err != nil {
				f.Error("getArticleInfo error:%s\noid:%s", err.Error(), oid)
				continue
			}
			//过滤无新id或type的文章,im为空的文章不过滤,当作无图文章来处理
			if artInfo[artFieldNewID] == "" || artInfo[artFieldType] == "" {
				continue
			}

			historyTmp[oid] = artInfo
			//过滤历史
			if _, ok := history[artInfo[artFieldNewID]]; ok {
				continue
			}
		}

		if flag & F_VERSION != 0 {
			if args.Version < version_cur {
				//低版本过滤组card
				if cardFlag {
					continue
				}
				//低版本过滤直播和图集
				if historyTmp[oid][artFieldType] == "14" || historyTmp[oid][artFieldType] == "15" {
					continue
				}
			}
		}
		if flag & F_SET_TOP != 0 {
			score, _ := strconv.ParseFloat(input[i + 1], 64)
			if score > 2000000000 {
				continue
			}
		}
		if (flag & F_HAS_PIC != 0)&&checkChanID(args.ChanID) {
			if cardFlag {
				continue
			}
			if historyTmp[oid][artFieldHasPic] == "1" {
				outputHasPic = append(outputHasPic, oid)
				countHasPic++
			} else {
				outputNorPic = append(outputNorPic, oid)
			}
			countTotal++
			if countTotal >= num&&countHasPic >= numHasPic {
				break
			}
		} else {
			output = append(output, input[i])
			if len(output) >= num {
				break
			}
		}
	}

	if countTotal > 0 {
		indexHasPic, indexNorPic := 0, 0
		for i := 0; i < num; i++ {
			if i % 2 == 0 {
				if indexHasPic < len(outputHasPic) {
					output = append(output, outputHasPic[indexHasPic])
					indexHasPic++
				} else if indexNorPic < len(outputNorPic) {
					output = append(output, outputNorPic[indexNorPic])
					indexNorPic++
				}
			}
			if i % 2 != 0 {
				if indexNorPic < len(outputNorPic) {
					output = append(output, outputNorPic[indexNorPic])
					indexNorPic++
				} else if indexHasPic < len(outputHasPic) {
					output = append(output, outputHasPic[indexHasPic])
					indexHasPic++
				}
			}
		}
	}

	if len(output) == 0 {
		f.Error("empty data after filter,args:%v", *args)
		//return nil, errors.New("empty data after filter")
	}

	if err := f.setChanHistory(output, historyTmp, args); err != nil {
		f.Error("recordHistory error:%s,args:%v", err.Error(), *args)
		return nil, errors.New("recordHistory failed")
	}

	var l int
	if len(input) < 30 {
		l = len(input)
	} else {
		l = 30
	}

	fmt.Printf("input:%v\n", input[:l])
	fmt.Printf("HasPic:%v\n", outputHasPic)
	fmt.Printf("NorPic:%v\n", outputNorPic)
	fmt.Printf("output:%v\n", output)

	return output, nil
}

func (f *Feed)setChanHistory(output []string, historyTmp map[string]map[string]string, args *ChanFeedGetArgs) error {
	rChanHisW, rChanHisKey, err := f.getChanHisRedis(args, CHAN_HIS_RD_W)
	if err != nil {
		return err
	}

	rChanR, err := f.Ctx.Redis("star_feed_r", 0)
	if err != nil {
		return err
	}

	//LPUSH
	for k := range output {
		//判断是否为组card
		if strings.Contains(output[k], ":") {
			newID := historyTmp[output[k]][artFieldNewID]
			if err := rChanHisW.LPush(rChanHisKey, newID); err != nil {
				f.Error("recordHistory error:%s,oid:%s", err.Error(), output[k])
				continue
			}
		} else {
			oidStr, err := rChanR.Get1(rCardPrefix + output[k])
			if err != nil {
				f.Error("get card oids error:%s,cardid:%s", err.Error(), output[k])
				continue
			}
			oidSlice := strings.Split(oidStr, ",")
			for _, oid := range oidSlice {
				if err := rChanHisW.LPush(rChanHisKey, historyTmp[oid][artFieldNewID]); err != nil {
					f.Error("recordHistory error:%s,oid:%s", err.Error(), oid)
					continue
				}
			}

		}
	}

	//LTRIM
	if err := rChanHisW.LTrim(rChanHisKey, 0, f.Ctx.Cfg.Feed.ChanFeedHistoryNum - 1); err != nil {
		return err
	}
	//EXPIRE
	if err := rChanHisW.Expire(rChanHisKey, f.Ctx.Cfg.Feed.ChanFeedHistoryDay * 24 * 60 * 60); err != nil {
		return err
	}
	return nil
}

func (f *Feed)getArticleInfo(oid string) (artInfo map[string]string, err error) {
	r, err := f.Ctx.Redis("r7018", 0)
	if err != nil {
		return
	}
	artInfo, err = r.HMGet1(rArticleInfoPrefix + oid, artFieldNewID, artFieldType, artFieldHasPic)
	return
}

func getChanMachineKey(chanID, subChanID string) string {
	if chanID == "1" {
		return "feed:1"
	}
	if chanID == "10015"&&subChanID != "" {
		return "feedu:10015:" + subChanID
	}
	return "feedu:" + chanID
}

func checkChanID(chanID string) bool {
	return chanID != "10015" && chanID != "1" && chanID != "10024"
}

func (f *Feed)getChanHisRedis(args *ChanFeedGetArgs, rwType int) (rd *utils.Redis, key string, err error) {
	var inputStr string
	if args.Uid != 0 {
		inputStr = strconv.FormatInt(args.Uid, 10)
	} else {
		inputStr = args.Imei
	}
	key = rChanHisPrefix + inputStr

	index := int(crc32.ChecksumIEEE([]byte(inputStr))) % 16
	if rwType == CHAN_HIS_RD_R {
		rd, err = f.Ctx.Redis("channel_feed_his_r", index)
	} else {
		rd, err = f.Ctx.Redis("channel_feed_his_w", index)
	}
	return
}

func (f *Feed)genID(bid int64) int64 {
	servers := f.Ctx.Cfg.Feed.IdGenServers
	if servers == nil || len(servers) == 0 {
		f.Error("genID:idGenServer empty")
		return 0
	}
	index := rand.Intn(len(servers))
	server := servers[index]

	url := fmt.Sprintf(server, bid)
	res, err := f.Ctx.HttpClient.Get(url)
	if err != nil {
		f.Error("genID:call idGenServer failed:%s", err.Error())
		return 0
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		f.Error("genID:read response failed:%s", err.Error())
		return 0
	}
	resM := make(map[string]interface{})
	decr := json.NewDecoder(strings.NewReader(string(body)))
	decr.UseNumber()
	if err = decr.Decode(&resM); err != nil {
		f.Error("genID:decode response failed:%s", err.Error())
		return 0
	}

	if errnoj, ok := resM["errno"].(json.Number); ok {
		if errno, _ := errnoj.Int64(); errno != 0 {
			f.Error("genID:response errno is invalid")
			return 0
		}
	}

	if idj, ok := resM["result"].(json.Number); ok {
		id, _ := idj.Int64()
		return id
	}
	return 0
}

func (f *Feed)decID(id int64) (m map[string]interface{}) {
	m = make(map[string]interface{})
	servers := f.Ctx.Cfg.Feed.IdDecServers
	if servers == nil || len(servers) == 0 {
		f.Error("decID:idDecServer empty")
		return
	}
	index := rand.Intn(len(servers))
	server := servers[index]
	url := fmt.Sprintf(server, id)
	res, err := f.Ctx.HttpClient.Get(url)
	if err != nil {
		f.Error("decID:call idDecServer failed:%s", err.Error())
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		f.Error("decID:read response failed:%s", err.Error())
		return
	}
	resM := make(map[string]interface{})
	decr := json.NewDecoder(strings.NewReader(string(body)))
	decr.UseNumber()
	if err = decr.Decode(&resM); err != nil {
		f.Error("decID:decode response failed:%s", err.Error())
		return
	}

	if errnoj, ok := resM["errno"].(json.Number); ok {
		if errno, _ := errnoj.Int64(); errno != 0 {
			f.Error("decID:response errno is invalid")
			return
		}
	}
	if result, ok := resM["result"].(map[string]interface{}); ok {
		m = result
	}
	return
}
