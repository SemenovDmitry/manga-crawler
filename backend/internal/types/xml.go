package types

// RSS структура RSS фида
type RSS struct {
	Channel Channel `xml:"channel"`
}

// Channel канал RSS фида
type Channel struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
	Items []Item `xml:"item"`
}

// Item элемент RSS фида (глава манги)
type Item struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
}
