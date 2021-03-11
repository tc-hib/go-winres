package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

const (
	errInvalidSet    = "invalid resource set definition"
	errInvalidCursor = "invalid cursor definition"
	errInvalidIcon   = "invalid icon definition"
)

type jsonDef map[string]map[string]map[string]interface{}

var typeIDToString = map[winres.ID]string{
	winres.RT_CURSOR:       "RT_CURSOR",
	winres.RT_BITMAP:       "RT_BITMAP",
	winres.RT_ICON:         "RT_ICON",
	winres.RT_MENU:         "RT_MENU",
	winres.RT_DIALOG:       "RT_DIALOG",
	winres.RT_STRING:       "RT_STRING",
	winres.RT_FONTDIR:      "RT_FONTDIR",
	winres.RT_FONT:         "RT_FONT",
	winres.RT_ACCELERATOR:  "RT_ACCELERATOR",
	winres.RT_RCDATA:       "RT_RCDATA",
	winres.RT_MESSAGETABLE: "RT_MESSAGETABLE",
	winres.RT_GROUP_CURSOR: "RT_GROUP_CURSOR",
	winres.RT_GROUP_ICON:   "RT_GROUP_ICON",
	winres.RT_VERSION:      "RT_VERSION",
	winres.RT_PLUGPLAY:     "RT_PLUGPLAY",
	winres.RT_VXD:          "RT_VXD",
	winres.RT_ANICURSOR:    "RT_ANICURSOR",
	winres.RT_ANIICON:      "RT_ANIICON",
	winres.RT_HTML:         "RT_HTML",
	winres.RT_MANIFEST:     "RT_MANIFEST",
}

var typeIDFromString = map[string]winres.ID{
	"RT_CURSOR":       winres.RT_CURSOR,
	"RT_BITMAP":       winres.RT_BITMAP,
	"RT_ICON":         winres.RT_ICON,
	"RT_MENU":         winres.RT_MENU,
	"RT_DIALOG":       winres.RT_DIALOG,
	"RT_STRING":       winres.RT_STRING,
	"RT_FONTDIR":      winres.RT_FONTDIR,
	"RT_FONT":         winres.RT_FONT,
	"RT_ACCELERATOR":  winres.RT_ACCELERATOR,
	"RT_RCDATA":       winres.RT_RCDATA,
	"RT_MESSAGETABLE": winres.RT_MESSAGETABLE,
	"RT_GROUP_CURSOR": winres.RT_GROUP_CURSOR,
	"RT_GROUP_ICON":   winres.RT_GROUP_ICON,
	"RT_VERSION":      winres.RT_VERSION,
	"RT_PLUGPLAY":     winres.RT_PLUGPLAY,
	"RT_VXD":          winres.RT_VXD,
	"RT_ANICURSOR":    winres.RT_ANICURSOR,
	"RT_ANIICON":      winres.RT_ANIICON,
	"RT_HTML":         winres.RT_HTML,
	"RT_MANIFEST":     winres.RT_MANIFEST,
}

func exportResources(dir string, rs *winres.ResourceSet, manifestInJSON bool) {
	res := jsonDef{}
	jsonName := filepath.Join(dir, "winres.json")

	rs.Walk(func(typeID, resID winres.Identifier, langID uint16, data []byte) bool {
		switch typeID {
		case winres.RT_CURSOR, winres.RT_ICON:
			return true
		}

		t, r, l := idsToStrings(typeID, resID, langID)
		filename := filepath.Join(dir, exportedName(res[t] == nil, data, typeID, resID, langID))

		if res[t] == nil {
			res[t] = make(map[string]map[string]interface{})
		}
		if res[t][r] == nil {
			res[t][r] = make(map[string]interface{})
		}

		printError := func(err error) {
			log.Printf("[%s][%s][%s] %v", t, r, l, err)
		}

		switch typeID {
		case winres.RT_GROUP_ICON:
			err := saveIcon(filename, rs, resID, langID)
			if err != nil {
				printError(err)
				return true
			}
			res[t][r][l] = filepath.Base(filename)
			return true
		case winres.RT_GROUP_CURSOR:
			err := saveCursor(filename, rs, resID, langID)
			if err != nil {
				printError(err)
				return true
			}
			res[t][r][l] = filepath.Base(filename)
			return true
		case winres.RT_BITMAP:
			err := saveBitmap(filename, data)
			if err != nil {
				printError(err)
				return true
			}
			res[t][r][l] = filepath.Base(filename)
			return true
		case winres.RT_VERSION:
			vi, err := version.FromBytes(data)
			if err != nil {
				printError(err)
				return true
			}
			res[t][r][l] = vi
			return true
		case winres.RT_MANIFEST:
			if manifestInJSON {
				m, err := winres.AppManifestFromXML(data)
				if err != nil {
					printError(err)
					return true
				}
				res[t][r][l] = m
				return true
			}
		}
		err := ioutil.WriteFile(filename, data, 0666)
		if err != nil {
			printError(err)
			return true
		}
		res[t][r][l] = filepath.Base(filename)
		return true
	})

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Println(err)
		return
	}

	err = ioutil.WriteFile(jsonName, b, 0666)
	if err != nil {
		log.Println(err)
	}
}

func saveIcon(filename string, rs *winres.ResourceSet, resID winres.Identifier, langID uint16) error {
	icon, err := rs.GetIconTranslation(resID, langID)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	err = icon.SaveICO(f)
	if err != nil {
		return err
	}

	return f.Close()
}

func saveCursor(filename string, rs *winres.ResourceSet, resID winres.Identifier, langID uint16) error {
	cursor, err := rs.GetCursorTranslation(resID, langID)
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	err = cursor.SaveCUR(f)
	if err != nil {
		return err
	}

	return f.Close()
}

func saveBitmap(filename string, dib []byte) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(dibToBMP(dib))
	if err != nil {
		return err
	}

	return f.Close()
}

func dibToBMP(dib []byte) []byte {
	dibHeader := struct {
		Size          uint32
		Width         int32
		Height        int32
		Planes        uint16
		BitCount      uint16
		Compression   uint32
		SizeImage     uint32
		XPelsPerMeter int32
		YPelsPerMeter int32
		ClrUsed       uint32
	}{}

	const (
		BI_PNG  = 4
		BI_JPEG = 5
	)

	err := binary.Read(bytes.NewReader(dib), binary.LittleEndian, &dibHeader)
	if err != nil || dibHeader.Size < 40 {
		return dib
	}

	bitsOffset := 14 + dibHeader.Size
	// https://docs.microsoft.com/en-us/previous-versions/dd183376(v=vs.85)
	// Several things don't seem to be true anymore:
	// BI_BITFIELD and ClrUsed don't seem to have any effect on 16bpp, 24bpp or 32bpp RGB bitmaps
	if dibHeader.Compression != BI_PNG && dibHeader.Compression != BI_JPEG {
		switch dibHeader.BitCount {
		case 1, 4, 8:
			if dibHeader.ClrUsed == 0 {
				dibHeader.ClrUsed = 1 << dibHeader.BitCount
			}
			bitsOffset += dibHeader.ClrUsed * 4
		case 16, 24, 32:
			// dibHeader.ClrUsed and dibHeader.Compression are ignored on purpose.
			// They don't seem to have any effect anymore.
			bitsOffset += dibHeader.ClrUsed * 4
		default:
			return dib
		}
	}

	fileSize := uint32(len(dib) + 14)
	bmp := make([]byte, 0, fileSize)
	buf := bytes.NewBuffer(bmp)
	buf.Write([]byte("BM"))
	binary.Write(buf, binary.LittleEndian, fileSize)
	buf.Write([]byte{0, 0, 0, 0})
	binary.Write(buf, binary.LittleEndian, bitsOffset)
	buf.Write(dib)

	return buf.Bytes()
}

func idsToStrings(typeID, resID winres.Identifier, langID uint16) (t, r, l string) {
	switch ident := typeID.(type) {
	case winres.ID:
		if s, ok := typeIDToString[ident]; ok {
			t = s
		} else {
			t = fmt.Sprintf("#%d", ident)
		}
	case winres.Name:
		t = fmt.Sprintf("%s", ident)
	}

	switch ident := resID.(type) {
	case winres.ID:
		r = fmt.Sprintf("#%d", ident)
	case winres.Name:
		r = fmt.Sprintf("%s", ident)
	}

	l = fmt.Sprintf("%04X", langID)

	return
}

func idsFromStrings(t, r, l string) (winres.Identifier, winres.Identifier, uint16, error) {
	var (
		typeID winres.Identifier
		resID  winres.Identifier
		langID uint16
	)

	if id, ok := typeIDFromString[t]; ok {
		typeID = id
	} else {
		typeID = stringToIdentifier(t)
	}
	if typeID == nil {
		return nil, nil, 0, errors.New("invalid type identifier")
	}

	resID = stringToIdentifier(r)
	if resID == nil {
		return nil, nil, 0, errors.New("invalid resource identifier")
	}

	n, err := strconv.ParseUint(l, 16, 16)
	if err != nil {
		return nil, nil, 0, errors.New("invalid language identifier")
	}
	langID = uint16(n)

	return typeID, resID, langID, nil
}

func stringToIdentifier(s string) winres.Identifier {
	if s == "" {
		return nil
	}
	if s[0] == '#' {
		n, err := strconv.ParseInt(s[1:], 10, 16)
		if err == nil {
			return winres.ID(n)
		}
	}
	return winres.Name(s)
}

func exportedName(first bool, data []byte, typeID, resID winres.Identifier, langID uint16) string {
	ext := "bin"
	t, r, l := idsToStrings(typeID, resID, langID)
	switch typeID {
	case winres.RT_MANIFEST:
		if resID == winres.ID(1) && langID == winres.LCIDDefault {
			return "app.manifest"
		}
		ext = "manifest"
		t = ""
	case winres.RT_GROUP_ICON:
		ext = "ico"
		t = ""
	case winres.RT_GROUP_CURSOR:
		ext = "cur"
		t = ""
	case winres.RT_BITMAP:
		ext = "bmp"
		t = ""
	case winres.RT_ANICURSOR:
		ext = "ani"
	case winres.RT_ANIICON:
		ext = "ani"
	case winres.RT_VERSION:
		if first {
			return "info.json"
		}
		t = "info"
		ext = "json"
	}
	if ext == "bin" {
		if len(data) > 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
			cfg, s, _ := image.DecodeConfig(bytes.NewReader(data))
			if s == "png" && cfg.ColorModel != nil && cfg.Width != 0 && cfg.Height != 0 {
				ext = "png"
			}
		}
	}
	if t == "" {
		return fmt.Sprintf("%s_%s.%s", r, l, ext)
	}
	return fmt.Sprintf("%s_%s_%s.%s", t, r, l, ext)
}

func importResources(rs *winres.ResourceSet, jsonName string) error {
	dir := filepath.Dir(jsonName)
	b, err := ioutil.ReadFile(jsonName)
	if err != nil {
		return err
	}

	res := jsonDef{}
	err = json.Unmarshal(b, &res)
	if err != nil {
		return err
	}

	for tid, t := range res {
		for _, r := range sortedRes(t) {
			for _, l := range sortedLang(r.langs) {
				typeID, resID, langID, err := idsFromStrings(tid, r.id, l.id)
				if err != nil {
					return err
				}
				switch typeID {
				case winres.RT_ICON:
					return errors.New("cannot import RT_ICON resources directly, use RT_GROUP_ICON instead")
				case winres.RT_CURSOR:
					return errors.New("cannot import RT_CURSOR resources directly, use RT_GROUP_CURSOR instead")
				case winres.RT_GROUP_CURSOR:
					cursor, err := loadCursor(dir, l.data)
					if err != nil {
						return err
					}
					err = rs.SetCursorTranslation(resID, langID, cursor)
					if err != nil {
						return err
					}
				case winres.RT_GROUP_ICON:
					icon, err := loadIcon(dir, l.data)
					if err != nil {
						return err
					}
					err = rs.SetIconTranslation(resID, langID, icon)
					if err != nil {
						return err
					}
				case winres.RT_VERSION:
					vi := version.Info{}
					j, _ := json.Marshal(l.data)
					err = json.Unmarshal(j, &vi)
					if err != nil {
						return err
					}
					rs.SetVersionInfo(vi)
				case winres.RT_BITMAP:
					filename, ok := l.data.(string)
					if !ok {
						return errors.New(errInvalidSet)
					}
					dib, err := loadBMP(filepath.Join(dir, filename))
					if err != nil {
						return err
					}
					err = rs.Set(winres.RT_BITMAP, resID, langID, dib)
					if err != nil {
						return err
					}
				case winres.RT_MANIFEST:
					switch val := l.data.(type) {
					case string:
						data, err := ioutil.ReadFile(filepath.Join(dir, val))
						if err != nil {
							return err
						}
						err = rs.Set(typeID, resID, langID, data)
						if err != nil {
							return err
						}
					default:
						j, _ := json.Marshal(val)
						m := winres.AppManifest{}
						err = json.Unmarshal(j, &m)
						if err != nil {
							return err
						}
						rs.SetManifest(m)
					}
				default:
					filename, ok := l.data.(string)
					if !ok {
						return errors.New(errInvalidSet)
					}
					data, err := ioutil.ReadFile(filepath.Join(dir, filename))
					if err != nil {
						return err
					}
					err = rs.Set(typeID, resID, langID, data)
					if err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

type resource struct {
	id    string
	langs map[string]interface{}
}

type lang struct {
	id   string
	data interface{}
}

func sortedRes(m map[string]map[string]interface{}) []resource {
	var res []resource
	for id, langs := range m {
		res = append(res, resource{id, langs})
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].id < res[j].id
	})
	return res
}

func sortedLang(m map[string]interface{}) []lang {
	var l []lang
	for id, data := range m {
		l = append(l, lang{id, data})
	}
	sort.Slice(l, func(i, j int) bool {
		return l[i].id < l[j].id
	})
	return l
}

func loadCursor(dir string, c interface{}) (*winres.Cursor, error) {
	switch c := c.(type) {
	case string:
		return loadCUR(filepath.Join(dir, c))

	case []interface{}:
		var images []winres.CursorImage
		for i := range c {
			o, ok := c[i].(map[string]interface{})
			if !ok {
				return nil, errors.New(errInvalidCursor)
			}
			curImg, err := loadCursorImage(dir, o)
			if err != nil {
				return nil, err
			}
			images = append(images, curImg)
		}
		return winres.NewCursorFromImages(images)

	case map[string]interface{}:
		curImg, err := loadCursorImage(dir, c)
		if err != nil {
			return nil, err
		}
		return winres.NewCursorFromImages([]winres.CursorImage{curImg})
	}

	return nil, errors.New(errInvalidCursor)
}

func loadCursorImage(dir string, c map[string]interface{}) (winres.CursorImage, error) {
	x, xOK := c["x"].(float64)
	y, yOK := c["y"].(float64)
	f, fOK := c["image"].(string)
	if !fOK || !xOK || !yOK {
		return winres.CursorImage{}, errors.New(errInvalidCursor)
	}

	img, err := loadImage(filepath.Join(dir, f))
	if err != nil {
		return winres.CursorImage{}, err
	}

	return winres.CursorImage{
		Image:   img,
		HotSpot: winres.HotSpot{X: uint16(x), Y: uint16(y)},
	}, nil
}

func loadIcon(dir string, x interface{}) (*winres.Icon, error) {
	switch x := x.(type) {
	case string:
		if strings.ToLower(filepath.Ext(x)) == ".ico" {
			return loadICO(filepath.Join(dir, x))
		}
		img, err := loadImage(filepath.Join(dir, x))
		if err != nil {
			return nil, err
		}
		return winres.NewIconFromResizedImage(img, nil)
	case []interface{}:
		var images []image.Image
		for i := range x {
			f, ok := x[i].(string)
			if !ok {
				return nil, errors.New(errInvalidIcon)
			}
			img, err := loadImage(filepath.Join(dir, f))
			if err != nil {
				return nil, err
			}
			images = append(images, img)
		}
		return winres.NewIconFromImages(images)
	}
	return nil, errors.New(errInvalidIcon)
}

func loadImage(name string) (image.Image, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func loadCUR(name string) (*winres.Cursor, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return winres.LoadCUR(f)
}

func loadICO(name string) (*winres.Icon, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return winres.LoadICO(f)
}

func loadBMP(name string) ([]byte, error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	if len(b) > 14 && b[0] == 'B' && b[1] == 'M' && int(b[5])<<24|int(b[4])<<16|int(b[3])<<8|int(b[2]) == len(b) {
		return b[14:], nil
	}

	return b, nil
}
