package analyze

import (
	"github.com/Rhymond/go-money"
	"github.com/ed-fx/go-soft4fx/internal/simulator"
	"github.com/pkg/errors"
	"math"
	"strconv"
	"time"
)

type Day struct {
	day time.Weekday

	NoOfTrades          int
	NoOfProfitTrades    int
	ProfitTradesInPips  float64
	ProfitTradesInMoney *money.Money
	NoOfLossTrades      int
	LossTradesInPips    float64
	LossTradesInMoney   *money.Money

	totalTradesDuration time.Duration
	simulatorGainPct    float64

	netProfit         *money.Money
	profitInPipsPct   float64
	lossInPipsPct     float64
	netGainInMoneyPct float64
}

type Weekday struct {
	monday    *Day
	tuesday   *Day
	wednesday *Day
	thursday  *Day
	friday    *Day
}

func (w Weekday) Monday() *Day {
	return w.monday
}

func (w Weekday) Tuesday() *Day {
	return w.tuesday
}

func (w Weekday) Wednesday() *Day {
	return w.wednesday
}

func (w Weekday) Thursday() *Day {
	return w.thursday
}

func (w Weekday) Friday() *Day {
	return w.friday
}

func (w *Weekday) getByDayOfWeek(weekday time.Weekday) *Day {
	switch weekday {
	case time.Monday:
		return w.monday
	case time.Tuesday:
		return w.tuesday
	case time.Wednesday:
		return w.wednesday
	case time.Thursday:
		return w.thursday
	case time.Friday:
		return w.friday
	}

	return nil
}

func (w Weekday) Days() []*Day {
	return []*Day{
		w.monday,
		w.tuesday,
		w.wednesday,
		w.thursday,
		w.friday,
	}
}

func (w *Weekday) append(order *simulator.Order) (err error) {
	openTime := order.OpenTime
	day := w.getByDayOfWeek(openTime.Weekday())
	if day == nil {
		err = errors.New("Invalid day for order [" + strconv.Itoa(order.Id) + "] Open time [" + openTime.String() + "]")
	} else {
		err = day.append(order)
	}

	return
}

func analyzeByWeekday(sim *simulator.Simulator) (weekday *Weekday, err error) {
	weekday = &Weekday{
		monday:    newDay(time.Monday),
		tuesday:   newDay(time.Tuesday),
		wednesday: newDay(time.Wednesday),
		thursday:  newDay(time.Thursday),
		friday:    newDay(time.Friday),
	}

	for _, order := range sim.ClosedOrders {
		if order.Type == simulator.Balance {
			continue
		}
		if err = weekday.append(order); err != nil {
			return
		}
	}

	netProfit := sim.Details.TotalNetProfit
	for _, day := range weekday.Days() {
		day.postConstruct(sim.ProfitInPips(), sim.LossInPips(), netProfit)
	}

	return
}

func newDay(day time.Weekday) *Day {
	return &Day{
		day:                day,
		ProfitTradesInPips: 0,
		LossTradesInPips:   0,
		simulatorGainPct:   0,
		profitInPipsPct:    0,
		lossInPipsPct:      0,
		netGainInMoneyPct:  0,
	}
}

func (d Day) Day() time.Weekday {
	return d.day
}

func (d Day) WinPct() float64 {
	if d.NoOfTrades == 0 {
		return 0
	}
	return math.Round(float64(d.NoOfProfitTrades)/float64(d.NoOfTrades)*100_00) / 100
}

func (d *Day) append(o *simulator.Order) (err error) {
	d.NoOfTrades++
	if o.IsWin() {
		d.NoOfProfitTrades++
		d.ProfitTradesInPips += o.ProfitPips()
		if d.ProfitTradesInMoney == nil {
			d.ProfitTradesInMoney = o.Profit
		} else {
			newProfit, _ := d.ProfitTradesInMoney.Add(o.Profit)
			d.ProfitTradesInMoney = newProfit
		}
	} else if o.IsLoss() {
		d.NoOfLossTrades++
		d.LossTradesInPips += o.ProfitPips()

		if d.LossTradesInMoney == nil {
			d.LossTradesInMoney = o.Profit
		} else {
			newLoss, _ := d.LossTradesInMoney.Add(o.Profit)
			d.LossTradesInMoney = newLoss
		}
	}

	d.totalTradesDuration += o.Duration()
	return
}

func (d *Day) postConstruct(simProfitInPips float64, simLossProfitInPips float64, netProfit *money.Money) {
	currencyCode := netProfit.Currency().Code
	d.ProfitTradesInMoney = initWithZeroMoney(d.ProfitTradesInMoney, currencyCode)
	d.LossTradesInMoney = initWithZeroMoney(d.LossTradesInMoney, currencyCode)

	d.netProfit = money.New(d.ProfitTradesInMoney.Amount()+d.LossTradesInMoney.Amount(), currencyCode)

	d.profitInPipsPct = pct(d.ProfitTradesInPips / simProfitInPips)
	if simLossProfitInPips != 0 {
		d.lossInPipsPct = pct(d.LossTradesInPips / simLossProfitInPips)
	}
	d.netGainInMoneyPct = pctInt64(d.netProfit.Amount(), netProfit.Amount())
}

func initWithZeroMoney(m *money.Money, currencyCode string) *money.Money {
	if m != nil {
		return m
	}

	return money.New(0, currencyCode)
}

func (d Day) ProfitInPipsPct() float64 {
	return d.profitInPipsPct
}

func (d Day) LossInPipsPct() float64 {
	return d.lossInPipsPct
}

func (d Day) NetProfitInMoney() *money.Money {
	return d.netProfit
}

func (d Day) NetGainInMoneyPct() float64 {
	return d.netGainInMoneyPct
}
