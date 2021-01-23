package main

import (
	"bytes"
	"crypto/md5"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

// language=json
const jsonRHM = `{
  "PNG": {
    "CUR-64X128": {
      "0000": "PNG_CUR-64X128_0000.png"
    }
  },
  "RT_BITMAP": {
    "ICON16": {
      "0000": "ICON16_0000.bmp"
    }
  },
  "RT_GROUP_CURSOR": {
    "CURSOR": {
      "0000": "CURSOR_0000.cur"
    }
  },
  "RT_GROUP_ICON": {
    "#1": {
      "0000": "#1_0000.ico"
    },
    "APP": {
      "0409": "APP_0409.ico",
      "040C": "APP_040C.ico"
    }
  },
  "RT_MANIFEST": {
    "#1": {
      "0000": {
        "identity": {},
        "description": "",
        "minimum-os": "win7",
        "execution-level": "",
        "ui-access": false,
        "auto-elevate": false,
        "dpi-awareness": "system",
        "disable-theming": false,
        "disable-window-filtering": false,
        "high-resolution-scrolling-aware": false,
        "ultra-high-resolution-scrolling-aware": false,
        "long-path-aware": false,
        "printer-driver-isolation": false,
        "gdi-scaling": false,
        "segment-heap": false,
        "use-common-controls-v6": false
      }
    }
  },
  "RT_VERSION": {
    "#1": {
      "0409": {
        "fixed": {
          "file_version": "1.2.3.4",
          "product_version": "5.6.7.8"
        },
        "info": {
          "0409": {
            "CompanyName": "WinRes Corp.",
            "FileDescription": "WinRes",
            "FileVersion": "1.2.3.4 (Build.42.42)",
            "InternalName": "winres",
            "LegalCopyright": "© WinRes.",
            "OriginalFilename": "WINRES.EXE",
            "ProductName": "WinRes CLI",
            "ProductVersion": "5.6.7.8"
          }
        }
      }
    }
  }
}`

// language=json
const jsonRH = `{
  "PNG": {
    "CUR-64X128": {
      "0000": "PNG_CUR-64X128_0000.png"
    }
  },
  "RT_BITMAP": {
    "ICON16": {
      "0000": "ICON16_0000.bmp"
    }
  },
  "RT_GROUP_CURSOR": {
    "CURSOR": {
      "0000": "CURSOR_0000.cur"
    }
  },
  "RT_GROUP_ICON": {
    "#1": {
      "0000": "#1_0000.ico"
    },
    "APP": {
      "0409": "APP_0409.ico",
      "040C": "APP_040C.ico"
    }
  },
  "RT_MANIFEST": {
    "#1": {
      "0000": "#1_0000.manifest"
    }
  },
  "RT_VERSION": {
    "#1": {
      "0409": {
        "fixed": {
          "file_version": "1.2.3.4",
          "product_version": "5.6.7.8"
        },
        "info": {
          "0409": {
            "CompanyName": "WinRes Corp.",
            "FileDescription": "WinRes",
            "FileVersion": "1.2.3.4 (Build.42.42)",
            "InternalName": "winres",
            "LegalCopyright": "© WinRes.",
            "OriginalFilename": "WINRES.EXE",
            "ProductName": "WinRes CLI",
            "ProductVersion": "5.6.7.8"
          }
        }
      }
    }
  }
}`

func Test_Extract(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	os.Args = []string{"./go-winres.exe", "extract", "--in", "_testdata/rh.exe", "--dir", "_testdata/tmp"}
	main()
	if loadJSON(t, "_testdata/tmp/winres.json") != jsonRHM {
		t.Error("json is different (json manifest)")
	}
	checkFile(t, "_testdata/tmp/#1_0000.ico", []byte{0x00, 0x24, 0x99, 0x66, 0x72, 0xaf, 0x62, 0x12, 0x35, 0x22, 0x9b, 0x7b, 0x2, 0x2f, 0xe5, 0x94})
	checkFile(t, "_testdata/tmp/APP_040C.ico", []byte{0xdd, 0xaf, 0x55, 0xb5, 0xd7, 0x3c, 0x65, 0xc3, 0x59, 0xb7, 0x3a, 0x39, 0x48, 0x62, 0xb3, 0x10})
	checkFile(t, "_testdata/tmp/APP_0409.ico", []byte{0xee, 0x29, 0x1d, 0xaa, 0xc1, 0x3d, 0x24, 0xfc, 0xc5, 0xe, 0x64, 0x71, 0x44, 0xf0, 0x1e, 0x46})
	checkFile(t, "_testdata/tmp/CURSOR_0000.cur", []byte{0x9f, 0x41, 0xe3, 0x26, 0xcc, 0x10, 0xb1, 0xed, 0xd9, 0x96, 0x38, 0x59, 0xc6, 0x93, 0x37, 0x9e})
	checkFile(t, "_testdata/tmp/PNG_CUR-64X128_0000.png", []byte{0x40, 0x83, 0x52, 0xdb, 0x5, 0xe7, 0xc2, 0x32, 0x5b, 0xc1, 0x40, 0x15, 0x5b, 0x16, 0x9f, 0x4d})
	checkFile(t, "_testdata/tmp/ICON16_0000.bmp", []byte{0xd7, 0x0c, 0xa6, 0x11, 0x23, 0xf6, 0x81, 0xef, 0xaa, 0xd3, 0x01, 0x61, 0x5f, 0x3e, 0x68, 0x1a})
	os.RemoveAll("_testdata/tmp")

	os.Args = []string{"./go-winres.exe", "extract", "--in", "_testdata/rh.exe", "--dir", "_testdata/tmp", "--xml-manifest"}
	main()
	if loadJSON(t, "_testdata/tmp/winres.json") != jsonRH {
		t.Error("json is different (xml manifest)")
	}
	checkFile(t, "_testdata/tmp/#1_0000.manifest", []byte{0x21, 0x19, 0x26, 0x2f, 0x8a, 0x5b, 0x5f, 0x64, 0x0e, 0x6e, 0xe7, 0x01, 0x20, 0xdb, 0xd2, 0x05})
	os.RemoveAll("_testdata/tmp")
}

func loadJSON(t *testing.T, file string) string {
	d, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(d)
}

func checkFile(t *testing.T, file string, hash []byte) {
	f, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h := md5.New()
	io.Copy(h, f)
	if !bytes.Equal(h.Sum(nil), hash) {
		t.Errorf("wrong hash for %s %x", file, h.Sum(nil))
	}
}
