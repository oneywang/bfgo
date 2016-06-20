package main

import . "github.com/sunwangme/bfgo/api/bfgateway"
import . "github.com/sunwangme/bfgo/api/bfdatafeed"

//********A.TBLR三状态*********
//1.状态就是计算当前价格属于上涨还是下跌、震荡。
//2.每个周期（日，小时，15分钟，3分钟）都有各自的状态。
//3.三状态具体计算算法：
//	上涨：60均线上，MACD红（线上红）
//	震荡：60均线下，MACD红（线下红）/60均线上，MACD绿（线上绿）
//	下跌：60均线下，MACD绿（线下绿）

//********B.TBLR九宫格*********
//	 慢	上涨   震荡   下跌
// 快
//上涨	多	  多	     空
//震荡	多	  -		 空
//下跌	多	  空	     空
//用法：
//1、快慢两个周期的三态生成九宫格，八格都有明确的交易方向
//2、目的是回避慢周期上大级别交易做反，实现震荡和上涨都可以轻松赚钱，扛一扛就可以赚了
//3、周线级别太大，而且可以体现在日线中；周线级别的上涨与下跌可以重仓参与，周线的意义更加在于发现大牛市/大熊市，然后加大杠杆大赚一把
//4、用两个级别，小时+日，两个都满足的时段，才可以交易（小时、日级别的交易做反，都可能爆仓），具体说就是日线和小时线的上涨、震荡状态才可以做多，下跌坚决不做多。
//5、用两个交易策略；一个策略只做多；一个策略只做空。两个策略的可交易时段是不同的，但双震荡过渡期有重叠
//6、波长和波幅除了交易，也可以在这里加以利用，比如日线/小时绿的前10根不做多，上一波的80%波长波幅内不操作
//7、非理性/非教条，想干就干，先拿起再放下，拿不起放不下；记得带套

//********C.TBLR交易策略*********
//“交易策略”由“状态构成的九宫格”决定。操作周期选择：
//60分钟震荡就是3分钟做（买卖都在3分钟），60分上涨就是15分做（买在3分钟，卖在15分钟）
//1.买卖操作：
//	买点：斜率向下背离红，斜率走平出金叉（30/60），斜率向上踩线红
//	加仓点（可全仓操作）：回踩红（最推荐）
//	止盈：三线止盈（破15/30/60均线，各清仓1/3）+死叉止盈+波长止盈+波幅止盈
//	止损：前低
//
//2.操作逻辑：
//	3分钟的震荡入场，赌震荡转上涨下跌，赌成功了就换成15分钟操作，
//	就是背离红开仓，踩线红加仓，然后：1)若大涨，60分钟明显突破，就赌赢了，用15分钟3线止盈；2)如果60分钟没什么改变，就在3分钟上3线止盈。
//	级别上是在3分钟是买或者卖，或者在15分钟是卖，15分钟上没有买
//	遇到大牛市了，3分钟频繁操作不好，得换大级别
//
//3.仓位：
//震荡：1/3仓位
//上涨：左侧：1/3；右侧：2/3

// 本策略的交易参数常量
const (
	TRADE_VOLUME int32  = 1
	VOLUME_LIMIT int32  = 5
	FAST_K_NUM   uint32 = 15
	SLOW_K_NUM   uint32 = 60
)

// 本策略的变量
var _period BfBarPeriod = BfBarPeriod_PERIOD_M01
var _historyBarsGot bool = false
var _barsCount uint32 = 0
var _currentBarMinute uint32 = 0
var _fastMa, _slowMa []float64
var _fastMa0, _fastMa1, _slowMa0, _slowMa1 float64 = 0, 0, 0, 0
var _positionLong, _positionShort int32 = 0, 0
var _pendingOrderIds []string

func initPosition(position *BfPositionData) {
	if _positionLong > 0 || _positionShort > 0 {
		// already inited
		return
	}
	if position.Direction == BfDirection_DIRECTION_LONG {
		_positionLong += position.Position
	} else if position.Direction == BfDirection_DIRECTION_SHORT {
		_positionShort += position.Position
	}
}

func updatePosition(direction BfDirection, offset BfOffset, volume int32) {
	if direction == BfDirection_DIRECTION_LONG && offset == BfOffset_OFFSET_OPEN {
		_positionLong += volume
	} else if direction == BfDirection_DIRECTION_LONG && offset == BfOffset_OFFSET_CLOSE {
		_positionLong -= volume
	} else if direction == BfDirection_DIRECTION_SHORT && offset == BfOffset_OFFSET_OPEN {
		_positionShort += volume
	} else if direction == BfDirection_DIRECTION_SHORT && offset == BfOffset_OFFSET_CLOSE {
		_positionShort -= volume
	}
}

func indexOf(a []string, v string) int {
	for i := range a {
		if a[i] == v {
			return i
		}
	}
	return -1
}

func without(a []string, v string) []string {
	var r []string
	j := 0
	for i := range a {
		if a[i] != v {
			r[j] = a[i]
			j++
		}
	}
	return r
}
