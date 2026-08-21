package stats

import (
	"math"
	"sort"
	"time"
)

var Windows = [...]int{7, 30, 90, 356}

type Price struct {
	Date      string
	BulkPrice *float64
}

type Value struct {
	Set   bool
	Value float64
}

type Result struct {
	MeanPrice       map[int]Value
	DiffMean        map[int]Value
	DiffMeanPct     map[int]Value
	StdDev          map[int]Value
	StdDevPct       map[int]Value
	Min             map[int]Value
	Max             map[int]Value
	Changes         map[int]Value
	MeanAll         Value
	DiffMeanAll     Value
	DiffMeanPctAll  Value
	MinAll          Value
	MinAllDate      string
	MaxAll          Value
	MaxAllDate      string
	ChangesAll      Value
}

func Calculate(prices []Price, today time.Time) Result {
	r := emptyResult()
	valid := make([]Price, 0, len(prices))
	for _, p := range prices {
		if p.BulkPrice == nil || *p.BulkPrice <= 0 || math.IsNaN(*p.BulkPrice) || math.IsInf(*p.BulkPrice, 0) {
			return r
		}
		if _, err := time.Parse("2006-01-02", p.Date); err != nil {
			return r
		}
		valid = append(valid, p)
	}
	if len(valid) == 0 {
		return r
	}
	sort.Slice(valid, func(i, j int) bool { return valid[i].Date < valid[j].Date })

	var allSum float64
	for i, p := range valid {
		v := *p.BulkPrice
		allSum += v
		if i == 0 || v <= r.MinAll.Value {
			r.MinAll = Value{true, v}; r.MinAllDate = p.Date
		}
		if i == 0 || v >= r.MaxAll.Value {
			r.MaxAll = Value{true, v}; r.MaxAllDate = p.Date
		}
		if i > 0 && v != *valid[i-1].BulkPrice { r.ChangesAll.Value++; r.ChangesAll.Set = true }
	}
	r.MeanAll = Value{true, allSum / float64(len(valid))}
	r.DiffMeanAll = Value{true, v(*valid[len(valid)-1].BulkPrice) - r.MeanAll.Value}
	if r.MeanAll.Value != 0 { r.DiffMeanPctAll = Value{true, r.DiffMeanAll.Value / r.MeanAll.Value * 100} }

	todayDate := today.Format("2006-01-02")
	for _, days := range Windows {
		cutoff := today.AddDate(0, 0, -days).Format("2006-01-02")
		var sum, sumSq float64
		count, changes := 0, 0
		var min, max, previous float64
		for _, p := range valid {
			if p.Date < cutoff || p.Date > todayDate { continue }
			x := *p.BulkPrice
			if count == 0 { min, max = x, x } else if x < min { min = x } else if x > max { max = x }
			if count > 0 && x != previous { changes++ }
			sum, sumSq, previous, count = sum+x, sumSq+x*x, x, count+1
		}
		if count == 0 { continue }
		mean := sum / float64(count)
		variance := sumSq/float64(count) - mean*mean
		if variance < 0 { variance = 0 }
		std := math.Sqrt(variance)
		r.MeanPrice[days] = Value{true, mean}
		r.DiffMean[days] = Value{true, *valid[len(valid)-1].BulkPrice - mean}
		if mean != 0 { r.DiffMeanPct[days] = Value{true, r.DiffMean[days].Value / mean * 100}; r.StdDevPct[days] = Value{true, std / mean * 100} }
		r.StdDev[days] = Value{true, std}
		r.Min[days], r.Max[days], r.Changes[days] = Value{true, min}, Value{true, max}, Value{true, float64(changes)}
	}
	return r
}

func v(x float64) float64 { return x }

func emptyResult() Result {
	return Result{MeanPrice: map[int]Value{}, DiffMean: map[int]Value{}, DiffMeanPct: map[int]Value{}, StdDev: map[int]Value{}, StdDevPct: map[int]Value{}, Min: map[int]Value{}, Max: map[int]Value{}, Changes: map[int]Value{}}
}
