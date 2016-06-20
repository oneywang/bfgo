package main

import (
	"log"
	"math"
	"strconv"
	"strings"
)
import . "github.com/sunwangme/bfgo/bftraderclient"
import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

//======
type DataRecorder struct {
	*BfTrderClient
	clientId     string
	tickHandler  bool
	tradeHandler bool
	logHandler   bool
	symbol       string
	exchange     string
}

// "%H:%M:%S.%f"==>"%H:%M:%S"
func ticktime2bartime(t string) string {
	var b string
	if dot := strings.LastIndex(t, "."); dot >= 0 {
		b = t[:dot]
	} else {
		log.Fatalf("Failed ticktime2bartime : %s", t)
	}
	return b
}

func bartime2minute(t string) int32 {
	var m int32
	if strings.Count(t, ":") != 2 {
		log.Fatalf("Failed bartime2minute : %s", t)
	}
	start := strings.Index(t, ":")
	stop := strings.LastIndex(t, ":")
	if stop > start {
		i, err := strconv.Atoi(t[start+1 : stop])
		if err != nil {
			log.Fatalf("Failed bartime2minute : %s, %v", t, err)
		} else {
			m = int32(i)
		}
	}
	return m
}

// 用Tick数据赋值Bar
func tick2bar(t *BfTickData, period BfBarPeriod, b *BfBarData) {
	b.Symbol = t.Symbol
	b.Exchange = t.Exchange
	b.Period = period

	b.ActionDate = t.ActionDate
	b.BarTime = ticktime2bartime(t.TickTime) //TODO: "%H:%M:%S.%f"==>"%H:%M:%S"
	b.Volume = t.Volume
	b.OpenInterest = t.OpenInterest
	b.LastVolume = t.LastVolume

	b.OpenPrice = t.LastPrice
	b.HighPrice = t.LastPrice
	b.LowPrice = t.LastPrice
	b.ClosePrice = t.LastPrice
}

// 不同品种当前的1分钟K线
var _bars_1min = make(map[string]BfBarData)

// 要把contract保存到datafeed里面才会看到数据
// 判断是否初始化了这个保存动作的标志
var _contract_inited = false

func insertContracts(client *DataRecorder) {
	resps, _ := client.GwGetContract(&BfGetContractReq{Symbol: "*", Exchange: "*"})
	for _, resp := range resps {
		client.InsertContract(resp)
	}
}

//======
func (client *DataRecorder) OnStart() {
	log.Printf("OnStart")
}
func (client *DataRecorder) OnTradeWillBegin(resp *BfNotificationData) {
	log.Printf("OnTradeWillBegin")
	log.Printf("%v", resp)
}

func (client *DataRecorder) OnGotContracts(resp *BfNotificationData) {
	log.Printf("OnGotContracts")
	log.Printf("%v", resp)

	//
	// save contracts
	//
	_contract_inited = true
	insertContracts(client)
}
func (client *DataRecorder) OnPing(resp *BfPingData) {
	log.Printf("OnPing")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnTick(tick *BfTickData) {
	//log.Printf("OnTick")
	//log.Printf("%v", tick)

	//
	// save tick
	//
	client.InsertTick(tick)

	// 要把contract保存到datafeed里面才会看到数据
	// ongotcontracts只有ctpgateway连接上ctp时候才发，所有盘中策略连接ctpgateway时候，是没有这个信息的。
	// 可以手工把ctpgateway ctp-stop然后ctp-start以下，就可以得到这个消息。我们这里程序自动判断如果没有调用则主动调用一次。
	if _contract_inited == false {
		_contract_inited = true
		insertContracts(client)
	}
	// 计算K线
	id := tick.Symbol + "@" + tick.Exchange
	// tickDatetime = datetime.strptime(tick.actionDate+tick.tickTime,"%Y%m%d%H:%M:%S.%f")

	bar, ok := _bars_1min[id]
	if !ok {
		bar = BfBarData{}
		tick2bar(tick, BfBarPeriod_PERIOD_M01, &bar)
		_bars_1min[id] = bar
		return
	}

	//print "update bar for: " + id
	if bartime2minute(ticktime2bartime(tick.TickTime)) != bartime2minute(bar.BarTime) {
		// 过去的一个bar存入datafeed
		log.Printf("Insert bar [%s]", tick.TickTime)
		log.Printf("%v", bar)
		client.InsertBar(&bar)

		// 初始化一个新的k线
		tick2bar(tick, BfBarPeriod_PERIOD_M01, &bar)
	} else {
		// 继续累加当前K线
		bar.HighPrice = math.Max(bar.HighPrice, tick.LastPrice)
		bar.LowPrice = math.Min(bar.LowPrice, tick.LastPrice)
		bar.ClosePrice = tick.LastPrice
		bar.Volume = tick.Volume
		bar.OpenInterest = tick.OpenInterest
		bar.LastVolume += tick.LastVolume
	}
	// 记得要赋值
	_bars_1min[id] = bar
}

func (client *DataRecorder) OnError(resp *BfErrorData) {
	log.Printf("OnError")
	log.Printf("%v", resp)

}
func (client *DataRecorder) OnLog(resp *BfLogData) {
	log.Printf("OnLog")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnTrade(resp *BfTradeData) {
	log.Printf("OnTrade")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnOrder(resp *BfOrderData) {
	log.Printf("OnOrder")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnPosition(resp *BfPositionData) {
	log.Printf("OnPosition")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnAccount(resp *BfAccountData) {
	log.Printf("OnAccount")
	log.Printf("%v", resp)
}
func (client *DataRecorder) OnStop() {
	log.Printf("OnStop")
}

//======
func main() {
	client := &DataRecorder{
		BfTrderClient: NewBfTraderClient(),
		clientId:      "Barsaver",
		tickHandler:   true,
		tradeHandler:  false,
		logHandler:    false,
		symbol:        "*", //rb1610",
		exchange:      "*"} //SHFE"}

	BfRun(client,
		client.clientId,
		client.tickHandler,
		client.tradeHandler,
		client.logHandler,
		client.symbol,
		client.exchange)
}
