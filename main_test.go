package main

import (
	"bytes"
	"crypto/md5"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

const tmpDir = "_testdata/tmp"

func Test_Extract(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	os.Args = []string{"./go-winres.exe", "extract", "--dir", "_testdata/tmp", "_testdata/rh.exe"}
	main()
	if loadJSON(t, "winres.json") != jsonRHM {
		t.Error("json is different (json manifest)")
	}
	checkFile(t, "#1_0000.ico", []byte{0x00, 0x24, 0x99, 0x66, 0x72, 0xaf, 0x62, 0x12, 0x35, 0x22, 0x9b, 0x7b, 0x2, 0x2f, 0xe5, 0x94})
	checkFile(t, "APP_040C.ico", []byte{0xdd, 0xaf, 0x55, 0xb5, 0xd7, 0x3c, 0x65, 0xc3, 0x59, 0xb7, 0x3a, 0x39, 0x48, 0x62, 0xb3, 0x10})
	checkFile(t, "APP_0409.ico", []byte{0xee, 0x29, 0x1d, 0xaa, 0xc1, 0x3d, 0x24, 0xfc, 0xc5, 0xe, 0x64, 0x71, 0x44, 0xf0, 0x1e, 0x46})
	checkFile(t, "CURSOR_0000.cur", []byte{0x9f, 0x41, 0xe3, 0x26, 0xcc, 0x10, 0xb1, 0xed, 0xd9, 0x96, 0x38, 0x59, 0xc6, 0x93, 0x37, 0x9e})
	checkFile(t, "PNG_CUR-64X128_0000.png", []byte{0x40, 0x83, 0x52, 0xdb, 0x5, 0xe7, 0xc2, 0x32, 0x5b, 0xc1, 0x40, 0x15, 0x5b, 0x16, 0x9f, 0x4d})
	checkFile(t, "ICON16_0000.bmp", []byte{0xd7, 0x0c, 0xa6, 0x11, 0x23, 0xf6, 0x81, 0xef, 0xaa, 0xd3, 0x01, 0x61, 0x5f, 0x3e, 0x68, 0x1a})
}

func Test_Init(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	func() {
		f := moveToTmpDir(t)
		defer f()

		os.Args = []string{"./go-winres.exe", "init"}
		main()

		s, err := os.Stat("winres/winres.json")
		if err != nil || s.Size() == 0 {
			t.Fatal(err)
		}
		s, err = os.Stat("winres/icon.png")
		if err != nil || s.Size() == 0 {
			t.Fatal(err)
		}
		s, err = os.Stat("winres/icon16.png")
		if err != nil || s.Size() == 0 {
			t.Fatal(err)
		}
	}()
}

func Test_Extract_XMLManifest(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	os.Args = []string{"./go-winres.exe", "extract", "--dir", "_testdata/tmp", "--xml-manifest", "_testdata/rh.exe"}
	main()
	if loadJSON(t, "winres.json") != jsonRH {
		t.Error("json is different (xml manifest)")
	}
	checkFile(t, "#1_0000.manifest", []byte{0x21, 0x19, 0x26, 0x2f, 0x8a, 0x5b, 0x5f, 0x64, 0x0e, 0x6e, 0xe7, 0x01, 0x20, 0xdb, 0xd2, 0x05})
	checkFile(t, "#1_0000.ico", []byte{0x00, 0x24, 0x99, 0x66, 0x72, 0xaf, 0x62, 0x12, 0x35, 0x22, 0x9b, 0x7b, 0x2, 0x2f, 0xe5, 0x94})
	checkFile(t, "APP_040C.ico", []byte{0xdd, 0xaf, 0x55, 0xb5, 0xd7, 0x3c, 0x65, 0xc3, 0x59, 0xb7, 0x3a, 0x39, 0x48, 0x62, 0xb3, 0x10})
	checkFile(t, "APP_0409.ico", []byte{0xee, 0x29, 0x1d, 0xaa, 0xc1, 0x3d, 0x24, 0xfc, 0xc5, 0xe, 0x64, 0x71, 0x44, 0xf0, 0x1e, 0x46})
	checkFile(t, "CURSOR_0000.cur", []byte{0x9f, 0x41, 0xe3, 0x26, 0xcc, 0x10, 0xb1, 0xed, 0xd9, 0x96, 0x38, 0x59, 0xc6, 0x93, 0x37, 0x9e})
	checkFile(t, "PNG_CUR-64X128_0000.png", []byte{0x40, 0x83, 0x52, 0xdb, 0x5, 0xe7, 0xc2, 0x32, 0x5b, 0xc1, 0x40, 0x15, 0x5b, 0x16, 0x9f, 0x4d})
	checkFile(t, "ICON16_0000.bmp", []byte{0xd7, 0x0c, 0xa6, 0x11, 0x23, 0xf6, 0x81, 0xef, 0xaa, 0xd3, 0x01, 0x61, 0x5f, 0x3e, 0x68, 0x1a})
}

func Test_Patch(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	copyFile(t, "_testdata/vs0.exe", "_testdata/tmp/temp.exe")
	os.Args = []string{"./go-winres.exe", "patch", "--in", "_testdata/icons.json", "--product-version", "1.2.3.4", "_testdata/tmp/temp.exe"}
	main()
	checkFile(t, "temp.exe", []byte{0x50, 0x55, 0x11, 0x9e, 0x38, 0x55, 0x83, 0x72, 0x3e, 0x62, 0xa1, 0xa2, 0x0a, 0xf5, 0xed, 0x5b})
	checkFile(t, "temp.exe.bak", []byte{0x29, 0x13, 0xa7, 0xc5, 0x4a, 0xf9, 0x47, 0xef, 0xd6, 0x4f, 0x37, 0xc5, 0x62, 0xba, 0xd4, 0x39})
}

func Test_Patch_Add(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	copyFile(t, "_testdata/rh.exe", "_testdata/tmp/temp.exe")
	os.Args = []string{"./go-winres.exe", "patch", "--in", "_testdata/icons.json", "--no-backup", "_testdata/tmp/temp.exe"}
	main()
	checkFile(t, "temp.exe", []byte{0x08, 0x05, 0x43, 0x1d, 0x55, 0x88, 0x65, 0x30, 0xb2, 0x1f, 0xea, 0xda, 0xe9, 0xf5, 0x3b, 0x4e})
	if _, err := os.Stat("_testdata/tmp/temp.exe.bak"); err == nil {
		t.Error("temp.exe.bak should not exist")
	}
}

func Test_Patch_Replace(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	copyFile(t, "_testdata/rh.exe", "_testdata/tmp/temp.exe")

	os.Args = []string{"./go-winres.exe", "patch", "--in", "_testdata/test.json", "--delete", "--file-version", "1.2", "_testdata/tmp/temp.exe"}
	main()
	checkFile(t, "temp.exe", []byte{0x60, 0xa6, 0x65, 0x1e, 0xfb, 0xca, 0x4f, 0x60, 0xde, 0x93, 0x0d, 0x78, 0x41, 0x3b, 0x45, 0x70})
	checkFile(t, "temp.exe.bak", []byte{0x79, 0xbc, 0x0f, 0x27, 0x2b, 0x3f, 0x82, 0x69, 0xfa, 0xc1, 0x42, 0x1d, 0xc7, 0xdb, 0x68, 0x6c})
}

func Test_GitTag(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	copyFile(t, "_testdata/vs0.exe", "_testdata/tmp/temp1.exe")
	copyFile(t, "_testdata/rh.exe", "_testdata/tmp/temp2.exe")

	func() {
		f := moveToTmpDir(t)
		defer f()

		createTmpGitTag(t, "v1.42.3.24")

		os.Args = []string{"./go-winres.exe", "patch", "--in", "../icons.json", "--file-version", "git-tag", "temp1.exe"}
		main()
		os.Args = []string{"./go-winres.exe", "patch", "--in", "../test.json", "--product-version", "git-tag", "temp2.exe"}
		main()
	}()

	checkFile(t, "temp1.exe", []byte{0x5f, 0x25, 0x38, 0x47, 0xd0, 0x40, 0x46, 0x76, 0x3a, 0x9f, 0xb8, 0x38, 0x6a, 0xf0, 0xed, 0xca})
	checkFile(t, "temp2.exe", []byte{0x08, 0xe6, 0x23, 0x4f, 0xd6, 0xb3, 0x0d, 0x85, 0x9c, 0x56, 0x5d, 0x3a, 0xb9, 0x6c, 0x05, 0x09})
}

func Test_Simply(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	os.Args = []string{
		"./go-winres.exe",
		"simply",
		"--arch",
		" amd64, 386 , arm, arm64 ",
		"--out",
		"_testdata/tmp/simply",
		"--product-version",
		"v42.5.3.8 alpha",
		"--file-version",
		"1.2.3.4.5.6.7",
		"--manifest",
		"gui",
		"--admin",
		"--file-description",
		"description",
		"--product-name",
		"Product name",
		"--copyright",
		"©",
		"--original-filename",
		"xxx.exe",
		"--icon",
		"_testdata/fr.ico",
	}
	main()
	checkFile(t, "simply_windows_arm64.syso", []byte{0xb1, 0x1b, 0x8f, 0x79, 0xea, 0xac, 0x72, 0xae, 0x0e, 0x8a, 0xa6, 0xf2, 0x2c, 0x61, 0x10, 0x8d})
	checkFile(t, "simply_windows_arm.syso", []byte{0x66, 0xe5, 0x83, 0xd1, 0x62, 0xe8, 0x51, 0x10, 0xb1, 0x2e, 0x5b, 0xdc, 0x57, 0x23, 0xd9, 0x8c})
	checkFile(t, "simply_windows_amd64.syso", []byte{0xc4, 0xfe, 0x6e, 0x34, 0xb2, 0xe5, 0x9e, 0x93, 0x00, 0xf7, 0x1f, 0x5a, 0x62, 0x9b, 0xea, 0x37})
	checkFile(t, "simply_windows_386.syso", []byte{0xa0, 0x02, 0x97, 0x0f, 0xc9, 0x0d, 0x2c, 0x28, 0xf1, 0x23, 0xd0, 0x31, 0x6b, 0x3f, 0x0d, 0x73})
}

func Test_SimplyGitTag(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	copyFile(t, "_testdata/cur-32x64.png", "_testdata/tmp/icon.png")

	func() {
		f := moveToTmpDir(t)
		defer f()

		createTmpGitTag(t, "v1.42.3.24")

		os.Args = []string{"./go-winres.exe", "simply", "--arch", "amd64", "--file-version", "git-tag", "--icon", "icon.png"}
		main()
		os.Args = []string{"./go-winres.exe", "simply", "--arch", "arm64", "--product-version", "git-tag", "--icon", "icon.png"}
		main()
	}()

	checkFile(t, "rsrc_windows_amd64.syso", []byte{0xcd, 0xc5, 0x45, 0x8c, 0x68, 0xe0, 0x00, 0x50, 0xda, 0x83, 0xe9, 0x4d, 0xf6, 0x0a, 0xba, 0x9f})
	checkFile(t, "rsrc_windows_arm64.syso", []byte{0x9f, 0xa4, 0x6c, 0x2a, 0x4d, 0xaf, 0xf8, 0xac, 0x61, 0x9e, 0x85, 0x3c, 0x50, 0x6a, 0xd8, 0xbe})
}

func Test_Simply_PNGIcon(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	os.Args = []string{
		"./go-winres.exe",
		"simply",
		"--out",
		"_testdata/tmp/rsrc.syso",
		"--arch",
		"386",
		"--no-suffix",
		"--icon",
		"icon.png",
	}
	main()
	checkFile(t, "rsrc.syso", []byte{0x8e, 0xc8, 0xe5, 0x44, 0x8e, 0x93, 0xd2, 0xab, 0x84, 0x10, 0x67, 0x16, 0xac, 0x70, 0x1a, 0xad})
}

func Test_Make(t *testing.T) {
	a := os.Args
	defer func() { os.Args = a }()

	f := makeTmpDir(t)
	defer f()

	os.Args = []string{
		"./go-winres.exe",
		"make",
		"--arch",
		" amd64, 386 , arm, arm64 ",
		"--out",
		"_testdata/tmp/make",
		"--product-version",
		"v42.5.3.8 alpha",
		"--file-version",
		"1.2.3.4.5.6.7",
		"--in",
		"_testdata/test.json",
	}
	main()
	checkFile(t, "make_windows_arm64.syso", []byte{0x8d, 0x2c, 0x92, 0x12, 0x56, 0xbc, 0xd0, 0x15, 0x9e, 0xde, 0x20, 0xee, 0x75, 0xed, 0xa6, 0x80})
	checkFile(t, "make_windows_arm.syso", []byte{0x75, 0x28, 0x81, 0x6d, 0x91, 0xf1, 0x6f, 0xee, 0x48, 0x7a, 0x8c, 0xba, 0x87, 0x8a, 0xd8, 0x6a})
	checkFile(t, "make_windows_amd64.syso", []byte{0x0d, 0xff, 0x1f, 0x03, 0x3d, 0xca, 0x14, 0x8e, 0x17, 0x9f, 0x3d, 0xb0, 0xb7, 0x52, 0x5f, 0x6c})
	checkFile(t, "make_windows_386.syso", []byte{0xe9, 0xf0, 0x06, 0x1d, 0xaf, 0xe2, 0x2e, 0xca, 0x45, 0xf1, 0x1e, 0x8d, 0x76, 0x29, 0xbe, 0x3c})
}

func copyFile(t *testing.T, src, dst string) {
	s, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	io.Copy(d, s)
}

func makeTmpDir(t *testing.T) func() {
	os.RemoveAll(tmpDir)

	err := os.MkdirAll(tmpDir, 0666)
	if err != nil {
		t.Fatal(err)
	}

	return func() {
		err = os.RemoveAll(tmpDir)
		if err != nil {
			t.Log(err)
		}
	}
}

func moveToTmpDir(t *testing.T) func() {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	return func() {
		err = os.Chdir(dir)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func createTmpGitTag(t *testing.T, tag string) {
	err := ioutil.WriteFile("tmp.txt", []byte{}, 0666)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "tmp.txt")
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "commit", "-m", ".")
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "tag", "-a", tag, "-m", ".")
	err = cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

func loadJSON(t *testing.T, file string) string {
	d, err := ioutil.ReadFile(filepath.Join(tmpDir, file))
	if err != nil {
		t.Fatal(err)
	}
	return string(d)
}

func checkFile(t *testing.T, file string, hash []byte) {
	f, err := os.Open(filepath.Join(tmpDir, file))
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
