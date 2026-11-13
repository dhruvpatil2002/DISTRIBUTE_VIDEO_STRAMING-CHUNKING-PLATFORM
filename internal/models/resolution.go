
package models

type Resolution struct {
	Name      string
	Width     string
	Height    string
	VideoBit  string
	AudioBit  string
	Bandwidth string
}

var Qualities = []Resolution{
	{
		Name:      "1080p",
		Width:     "1920",
		Height:    "1080",
		VideoBit:  "5000k",
		AudioBit:  "192k",
		Bandwidth: "5500000",
	},
	{
		Name:      "720p",
		Width:     "1280",
		Height:    "720",
		VideoBit:  "2500k",
		AudioBit:  "128k",
		Bandwidth: "2800000",
	},
	{
		Name:      "480p",
		Width:     "854",
		Height:    "480",
		VideoBit:  "1000k",
		AudioBit:  "96k",
		Bandwidth: "1200000",
	},
}
