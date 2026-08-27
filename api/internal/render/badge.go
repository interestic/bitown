package render

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/interestic/bitown/internal/citycore"
)

// badgeTmpl is the shields.io-style city badge. html/template escapes Name so
// a city title cannot break out of the SVG text nodes.
var badgeTmpl = template.Must(template.New("badge").Parse(`<svg xmlns="http://www.w3.org/2000/svg" width="160" height="20">
  <linearGradient id="s" x2="0" y2="100%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <rect rx="3" width="160" height="20" fill="#555"/>
  <rect rx="3" x="80" width="80" height="20" fill="#4c9e4c"/>
  <rect rx="3" width="160" height="20" fill="url(#s)"/>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="40" y="15" fill="#010101" fill-opacity=".3">{{.Name}}</text>
    <text x="40" y="14">{{.Name}}</text>
    <text x="120" y="15" fill="#010101" fill-opacity=".3">pop {{.Pop}}</text>
    <text x="120" y="14">pop {{.Pop}}</text>
  </g>
</svg>`))

type badgeView struct {
	Name string
	Pop  int
}

// BuildCityBadgeSVG renders the README / share badge for a city.
func BuildCityBadgeSVG(city *citycore.City) ([]byte, error) {
	if city == nil {
		return nil, fmt.Errorf("city is nil")
	}
	var buf bytes.Buffer
	if err := badgeTmpl.Execute(&buf, badgeView{Name: city.Name, Pop: city.Pop.Int()}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
