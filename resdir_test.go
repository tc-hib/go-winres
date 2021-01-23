package main

import (
	"bytes"
	"image"
	"image/png"
	"reflect"
	"testing"

	"github.com/tc-hib/winres"
)

func Test_exportedName(t *testing.T) {
	var (
		pngBuf  bytes.Buffer
		fakePng = []byte{0x89, 'P', 'N', 'G', 0xD, 0xA, 0x1A, 0xA, 0xFF}
	)

	img := image.NewNRGBA(image.Rectangle{Min: image.Point{0, 0}, Max: image.Point{1, 1}})
	png.Encode(&pngBuf, img)

	type args struct {
		first  bool
		data   []byte
		typeID winres.Identifier
		resID  winres.Identifier
		langID uint16
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{true, fakePng, winres.Name("PNG"), winres.ID(1), 0x409},
			want: "PNG_#1_0409.bin",
		},
		{
			args: args{true, pngBuf.Bytes(), winres.Name("PNG"), winres.ID(1), 0x409},
			want: "PNG_#1_0409.png",
		},
		{
			args: args{true, pngBuf.Bytes(), winres.RT_MANIFEST, winres.ID(1), 0x409},
			want: "app.manifest",
		},
		{
			args: args{false, pngBuf.Bytes(), winres.RT_MANIFEST, winres.ID(1), 0x40C},
			want: "#1_040C.manifest",
		},
		{
			args: args{true, pngBuf.Bytes(), winres.RT_VERSION, winres.Name("Hey"), 0x409},
			want: "info.json",
		},
		{
			args: args{false, pngBuf.Bytes(), winres.RT_VERSION, winres.Name("Hey"), 0x409},
			want: "info_Hey_0409.json",
		},
		{
			args: args{true, pngBuf.Bytes(), winres.RT_RCDATA, winres.Name("Hey"), 0x409},
			want: "RT_RCDATA_Hey_0409.png",
		},
		{
			args: args{true, nil, winres.RT_RCDATA, winres.Name("Hey"), 0x409},
			want: "RT_RCDATA_Hey_0409.bin",
		},
		{
			args: args{true, nil, winres.ID(42), winres.Name("Hey"), 0x409},
			want: "#42_Hey_0409.bin",
		},
		{
			args: args{false, pngBuf.Bytes(), winres.RT_BITMAP, winres.Name("Hey"), 0x409},
			want: "Hey_0409.bmp",
		},
		{
			args: args{false, pngBuf.Bytes(), winres.RT_ANIICON, winres.ID(42), 0x409},
			want: "RT_ANIICON_#42_0409.ani",
		},
		{
			args: args{true, pngBuf.Bytes(), winres.RT_ANICURSOR, winres.ID(42), 0x409},
			want: "RT_ANICURSOR_#42_0409.ani",
		},
		{
			args: args{false, pngBuf.Bytes(), winres.RT_GROUP_CURSOR, winres.ID(42), 0x401},
			want: "#42_0401.cur",
		},
		{
			args: args{true, pngBuf.Bytes(), winres.RT_GROUP_ICON, winres.Name("APPICON"), 0x402},
			want: "APPICON_0402.ico",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := exportedName(tt.args.first, tt.args.data, tt.args.typeID, tt.args.resID, tt.args.langID); got != tt.want {
				t.Errorf("exportedName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_idsFromStrings(t *testing.T) {
	type args struct {
		t string
		r string
		l string
	}
	type want struct {
		t   winres.Identifier
		r   winres.Identifier
		l   uint16
		err bool
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			args: args{"RT_ICON", "RT_ICON", "RT_ICON"},
			want: want{nil, nil, 0, true},
		},
		{
			args: args{"RT_ICON", "#1", "#1"},
			want: want{nil, nil, 0, true},
		},
		{
			args: args{"#1", "#1", "040C"},
			want: want{winres.ID(1), winres.ID(1), 0x40C, false},
		},
		{
			args: args{"#24", "Hello", "FFFF"},
			want: want{winres.RT_MANIFEST, winres.Name("Hello"), 0xFFFF, false},
		},
		{
			args: args{"#24", "Hello", "1FFFF"},
			want: want{nil, nil, 0, true},
		},
		{
			args: args{"é", "é", "é"},
			want: want{nil, nil, 0, true},
		},
		{
			args: args{"é", "é", "0"},
			want: want{winres.Name("é"), winres.Name("é"), 0, false},
		},
		{
			args: args{"A", "", "0"},
			want: want{nil, nil, 0, true},
		},
		{
			args: args{"", "A", "0"},
			want: want{nil, nil, 0, true},
		},
		{
			args: args{"##1", "1", "0"},
			want: want{winres.Name("##1"), winres.Name("1"), 0, false},
		},
		{
			args: args{"1", "##1", "0"},
			want: want{winres.Name("1"), winres.Name("##1"), 0, false},
		},
		{
			// Fails later in winres
			args: args{"\x00", "A", "42"},
			want: want{winres.Name("\x00"), winres.Name("A"), 0x42, false},
		},
		{
			// Fails later in winres
			args: args{"A", "\x00", "42"},
			want: want{winres.Name("A"), winres.Name("\x00"), 0x42, false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotT, gotR, gotL, err := idsFromStrings(tt.args.t, tt.args.r, tt.args.l)
			got := want{gotT, gotR, gotL, err != nil}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("idsFromStrings() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_dibToBMP(t *testing.T) {
	type args struct {
		dib []byte
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{
			name: "1bpp-1",
			args: args{[]byte{40, 0x04: 1, 0x08: 1, 0x0C: 1, 0x0E: 1, 0x20: 1, 0x2F: 0}},
			want: 0x3A,
		},
		{
			name: "1bpp-0",
			args: args{[]byte{40, 0x04: 1, 0x08: 1, 0x0C: 1, 0x0E: 1, 0x20: 0, 0x33: 0}},
			want: 0x3E,
		},
		{
			name: "4bpp-5",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 4, 0x20: 5, 0x7B: 0}},
			want: 0x4A,
		},
		{
			name: "4bpp-0",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 4, 0x20: 0, 0xA7: 0}},
			want: 0x76,
		},
		{
			name: "8bpp-25",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 8, 0x20: 25, 0xB3: 0}},
			want: 0x9A,
		},
		{
			name: "8bpp-0",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 8, 0x20: 0, 0x44F: 0}},
			want: 0x436,
		},
		{
			name: "16bpp",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 16, 0x63: 0}},
			want: 0x36,
		},
		{
			name: "24bpp",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 24, 0x77: 0}},
			want: 0x36,
		},
		{
			name: "32bpp",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 32, 0x95: 0}},
			want: 0x36,
		},
		{
			name: "2bpp",
			args: args{[]byte{40, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 2, 0x95: 0}},
			want: 0,
		},
		{
			name: "not dib",
			args: args{[]byte{10, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 2, 0x95: 0}},
			want: 0,
		},
		{
			name: "eof",
			args: args{[]byte{10, 0x04: 5, 0x08: 5, 0x0C: 1, 0x0E: 1}},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bmp := dibToBMP(tt.args.dib)
			if tt.want != 0 || !bytes.Equal(bmp, tt.args.dib) {
				size := uint32(bmp[2]) | uint32(bmp[3])<<8 | uint32(bmp[4])<<16 | uint32(bmp[5])<<24
				offset := uint32(bmp[10]) | uint32(bmp[11])<<8 | uint32(bmp[12])<<16 | uint32(bmp[13])<<24
				wantedSize := uint32(14 + len(tt.args.dib))
				if size != wantedSize || offset != tt.want {
					t.Errorf("dibToBMP() = (%v, %v), want (%v, %v)", size, offset, wantedSize, tt.want)
				}
			}
		})
	}
}
