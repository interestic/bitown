package citycore

import "math"

// Metrics are Core.hx-style derived indicators for a city.
// Percent fields are integers 0–100. Income is absolute currency units.
type Metrics struct {
	Income        int `json:"income"`
	Unemployment  int `json:"unemployment"`  // % (higher = worse)
	Roads         int `json:"roads"`         // % capacity (higher = better)
	Pollution     int `json:"pollution"`     // % (higher = worse)
	Crime         int `json:"crime"`         // % (higher = worse)
}

// ComputeMetrics implements Miniville/com/Core.hx formulas.
//
//	income      = int( (pop*1.5 + ind*2.5 + com*4) * 100 )
//	失業係数     = 1 - min( (ind*3 + com + 50) / pop , 1 )
//	道路係数     = min( (tra*5 + 100) / pop , 1 )
//	汚染係数     = clamp01( ( (ind + pop*0.1 + com*0.3) - (env*2 + 200) ) / ind )
//	犯罪係数     = 1 - min( (sec*4 + 300) / pop , 1 )
func ComputeMetrics(c *City) Metrics {
	if c == nil {
		return Metrics{}
	}
	pop := float64(c.Pop.Int())
	ind := float64(c.Ind.Int())
	tra := float64(c.Tra.Int())
	sec := float64(c.Sec.Int())
	env := float64(c.Env.Int())
	com := float64(c.Com.Int())

	income := int(math.Floor((pop*1.5 + ind*2.5 + com*4) * 100))

	unemp := 0.0
	roads := 1.0
	crime := 0.0
	if pop > 0 {
		unemp = 1 - math.Min((ind*3+com+50)/pop, 1)
		roads = math.Min((tra*5+100)/pop, 1)
		crime = 1 - math.Min((sec*4+300)/pop, 1)
	}

	pollution := 0.0
	if ind > 0 {
		pollution = clamp01(((ind + pop*0.1 + com*0.3) - (env*2 + 200)) / ind)
	}

	return Metrics{
		Income:       income,
		Unemployment: pct(unemp),
		Roads:        pct(roads),
		Pollution:    pct(pollution),
		Crime:        pct(crime),
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func pct(coef float64) int {
	return int(math.Round(clamp01(coef) * 100))
}
