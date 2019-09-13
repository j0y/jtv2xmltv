package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"golang.org/x/text/encoding/charmap"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please execute with filename as first parameter")
		return
	}
	// Open a zip archive for reading.
	r, err := zip.OpenReader(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	tv := Tv{}
	// Iterate through the files in the archive,
	// printing some of their contents.
	channelID := 0
	for _, f := range r.File {
		extension := f.Name[len(f.Name)-4:]
		if extension != ".pdt" {
			continue
		}
		channelID++
		file := f.Name[0 : len(f.Name)-4]
		channelName := file
		chand := Channel{DisplayName: channelName}
		chand.ID = strconv.Itoa(channelID)
		tv.Channel = append(tv.Channel, chand)

		filerc, err := f.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer filerc.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(filerc)
		contents := buf.String()
		channelTitles := parseTitles(contents)

		schedulesFile := fmt.Sprintf("%s.ndx", file)
		var channelSchedules []time.Time
		for _, ff := range r.File {
			if ff.Name != schedulesFile {
				continue
			}

			schedFilerc, err := ff.Open()
			if err != nil {
				log.Fatal(err)
			}
			defer schedFilerc.Close()

			schedBuf := new(bytes.Buffer)
			schedBuf.ReadFrom(schedFilerc)
			schedContents := schedBuf.String()
			channelSchedules = parseSchedule(schedContents)
		}
		i := 0
		for _, curTitle := range channelTitles {
			if i < len(channelSchedules)-1 {
				start := channelSchedules[i]
				stop := channelSchedules[i+1]
				progr := Programme{Channel: strconv.Itoa(channelID), Start: string(start.Format("20060102150405")), Stop: string(stop.Format("20060102150405")), Title: curTitle}
				tv.Programme = append(tv.Programme, progr)
				i++
			}
		}

		channelTitles = append(channelTitles, "")

		filerc.Close()
	}

	output := os.Stdout
	xmlWriter := io.Writer(output)
	xmlWriter.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n"))

	enc := xml.NewEncoder(xmlWriter)
	enc.Indent("", "    ")
	if err := enc.Encode(tv); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}

/*
pdt format:
26 header symbols + 0A 0A 0A
2 length bytes (x17 x00 = 23)
program name (Телеканал "Доброе утро" - 23 bytes in CP1251)
*/

func parseTitles(data string) []string {
	jtvHeader := "JTV 3.x TV Program Data\x0a\x0a\x0a"
	if data[0:26] != jtvHeader {
		panic("Invalid JTV format")
	}
	data = data[26:]
	var titles []string

	for len(data) > 26 {
		var titleLength int
		titleLength = int(binary.LittleEndian.Uint16([]byte(data[0:2])))
		data = data[2:]
		title := data[0:titleLength]
		dec := charmap.Windows1251.NewDecoder()
		title, _ = dec.String(title)
		titles = append(titles, title)
		data = data[titleLength:]
	}

	return titles
}

/*
ndx:
first two bytes is record count
12 bytes records:
   * First two bytes is always 0
   * Eight bytes of FILETIME structure (Contains a 64-bit value representing the number of
                                        100-nanosecond intervals since January 1, 1601 (UTC).)
   * Two bytes - the offset pointer to TV-show characters number title in .pdt file.
*/
func parseSchedule(data string) []time.Time {
	var schedules []time.Time
	recordsNum := int(binary.LittleEndian.Uint16([]byte(data[0:2])))
	data = data[2:]
	i := 0
	for i < recordsNum {
		i++
		record := data[0:12]
		data = data[12:]
		datetime := filetimeToDatetime(record[2 : len(record)-2])
		schedules = append(schedules, datetime)
	}

	return schedules
}

func filetimeToDatetime(dtime string) time.Time {
	filetime := int64(binary.LittleEndian.Uint64([]byte(dtime)))
	timestamp := getTime(filetime)

	return timestamp
}

/*
cheating time.Duration 290 years limit
*/
func getTime(input int64) time.Time {
	t := time.Date(1601, 1, 1, 0, 0, 0, 0, time.UTC)
	d := time.Duration(input)
	for i := 0; i < 100; i++ {
		t = t.Add(d)
	}
	return t
}

// Channel individual channel
type Channel struct {
	XMLName     xml.Name `xml:"channel"`
	Text        string   `xml:",chardata"`
	ID          string   `xml:"id,attr"`
	DisplayName string   `xml:"display-name"`
}

// Programme tv programs
type Programme struct {
	XMLName xml.Name `xml:"programme"`
	Text    string   `xml:",chardata"`
	Channel string   `xml:"channel,attr"`
	Start   string   `xml:"start,attr"`
	Stop    string   `xml:"stop,attr"`
	Title   string   `xml:"title"`
}

// Tv for xml export
type Tv struct {
	XMLName   xml.Name `xml:"tv"`
	Text      string   `xml:",chardata"`
	Channel   []Channel
	Programme []Programme
}
