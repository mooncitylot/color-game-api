package models

// ColorAPIResponse represents the response from thecolorapi.com
type ColorAPIResponse struct {
	Mode   string  `json:"mode"`
	Count  string  `json:"count"`
	Colors []Color `json:"colors"`
	Seed   Seed    `json:"seed"`
}

// Color represents a single color in the palette
type Color struct {
	Hex      ColorHex      `json:"hex"`
	RGB      ColorRGB      `json:"rgb"`
	HSL      ColorHSL      `json:"hsl"`
	HSV      ColorHSV      `json:"hsv"`
	Name     ColorName     `json:"name"`
	CMYK     ColorCMYK     `json:"cmyk"`
	XYZ      ColorXYZ      `json:"XYZ"`
	Image    ColorImage    `json:"image"`
	Contrast ColorContrast `json:"contrast"`
	Links    ColorLinks    `json:"_links"`
	Embedded ColorEmbedded `json:"_embedded"`
}

// Seed represents the seed color used to generate the palette
type Seed struct {
	Hex   ColorHex   `json:"hex"`
	RGB   ColorRGB   `json:"rgb"`
	HSL   ColorHSL   `json:"hsl"`
	HSV   ColorHSV   `json:"hsv"`
	Name  ColorName  `json:"name"`
	CMYK  ColorCMYK  `json:"cmyk"`
	XYZ   ColorXYZ   `json:"XYZ"`
	Image ColorImage `json:"image"`
	Links ColorLinks `json:"_links"`
}

type ColorHex struct {
	Value string `json:"value"`
	Clean string `json:"clean"`
}

type ColorRGB struct {
	Fraction Fraction `json:"fraction"`
	R        int      `json:"r"`
	G        int      `json:"g"`
	B        int      `json:"b"`
	Value    string   `json:"value"`
}

type Fraction struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

type ColorHSL struct {
	Fraction FractionHSL `json:"fraction"`
	H        int         `json:"h"`
	S        int         `json:"s"`
	L        int         `json:"l"`
	Value    string      `json:"value"`
}

type FractionHSL struct {
	H float64 `json:"h"`
	S float64 `json:"s"`
	L float64 `json:"l"`
}

type ColorHSV struct {
	Fraction FractionHSV `json:"fraction"`
	Value    string      `json:"value"`
	H        int         `json:"h"`
	S        int         `json:"s"`
	V        int         `json:"v"`
}

type FractionHSV struct {
	H float64 `json:"h"`
	S float64 `json:"s"`
	V float64 `json:"v"`
}

type ColorName struct {
	Value           string `json:"value"`
	ClosestNamedHex string `json:"closest_named_hex"`
	ExactMatchName  bool   `json:"exact_match_name"`
	Distance        int    `json:"distance"`
}

type ColorCMYK struct {
	Fraction FractionCMYK `json:"fraction"`
	Value    string       `json:"value"`
	C        int          `json:"c"`
	M        int          `json:"m"`
	Y        int          `json:"y"`
	K        int          `json:"k"`
}

type FractionCMYK struct {
	C float64 `json:"c"`
	M float64 `json:"m"`
	Y float64 `json:"y"`
	K float64 `json:"k"`
}

type ColorXYZ struct {
	Fraction FractionXYZ `json:"fraction"`
	Value    string      `json:"value"`
	X        int         `json:"X"`
	Y        int         `json:"Y"`
	Z        int         `json:"Z"`
}

type FractionXYZ struct {
	X float64 `json:"X"`
	Y float64 `json:"Y"`
	Z float64 `json:"Z"`
}

type ColorImage struct {
	Bare  string `json:"bare"`
	Named string `json:"named"`
}

type ColorContrast struct {
	Value string `json:"value"`
}

type ColorLinks struct {
	Self Link `json:"self"`
}

type Link struct {
	Href string `json:"href"`
}

type ColorEmbedded struct {
	// This can be expanded if needed
}
