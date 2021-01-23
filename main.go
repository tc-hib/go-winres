package main

import (
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "embed"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
	"github.com/urfave/cli/v2"
)

const (
	defaultJSONFile   = "winres/winres.json"
	defaultIconFile   = "winres/icon.png"
	defaultIcon16File = "winres/icon16.png"
	defaultOutDir     = "winres"
	defaultArch       = "amd64,386"

	flagArch        = "arch"
	flagOutput      = "out"
	flagOutputDir   = "dir"
	flagInput       = "in"
	flagNoSuffix    = "no-suffix"
	flagNoBackup    = "no-backup"
	flagDelete      = "delete"
	flagXMLManifest = "xml-manifest"

	flagProductVersion = "product-version"
	flagFileVersion    = "file-version"

	flagInfoDescription = "file-description"
	flagInfoProductName = "product-name"
	flagInfoCopyright   = "copyright"
	flagInfoFilename    = "original-filename"

	flagIconFile     = "icon"
	flagRequireAdmin = "admin"
	flagManifest     = "manifest"

	manifestNone = "none"
	manifestCLI  = "cli"
	manifestGUI  = "gui"
)

//go:generate go run github.com/tc-hib/go-winres make

func main() {
	versionFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  flagProductVersion,
			Usage: "set product version (overrides the json file)",
		},
		&cli.StringFlag{
			Name:  flagFileVersion,
			Usage: "set file version (overrides the json file)",
		},
	}

	commonMakeFlags := append([]cli.Flag{
		&cli.StringFlag{
			Name:    flagArch,
			Usage:   "comma separated list of target architectures such as amd64,386,arm,arm64",
			Value:   defaultArch,
			EnvVars: []string{"GOARCH"},
		},
	}, versionFlags...)

	app := cli.App{
		Name:        "go-winres",
		Description: "A tool for embedding resources in Windows executables",
		Commands: []*cli.Command{
			{
				Name:   "init",
				Usage:  "Create an initial ./winres/winres.json",
				Action: cmdInit,
			},
			{
				Name:   "make",
				Usage:  "Make syso files for the \"go build\" command",
				Action: cmdMake,
				Flags: append([]cli.Flag{
					&cli.StringFlag{
						Name:      flagInput,
						Usage:     "name of the input json file",
						Value:     defaultJSONFile,
						TakesFile: true,
					},
					&cli.StringFlag{
						Name:  flagOutput,
						Usage: "name or prefix of the object file (syso)",
						Value: "rsrc",
					},
					&cli.BoolFlag{
						Name:  flagNoSuffix,
						Usage: "don't add target suffixes such as \"_windows_386\"",
						Value: false,
					},
				},
					commonMakeFlags...),
			},
			{
				Name:   "simply",
				Usage:  "Make syso files for the \"go build\" command (simplified)",
				Action: cmdSimply,
				Flags: append(commonMakeFlags, []cli.Flag{
					&cli.StringFlag{
						Name:  flagManifest,
						Value: manifestCLI,
						Usage: "type of manifest: \"cli\", \"gui\" or \"none\"",
					},
					&cli.BoolFlag{
						Name:  flagRequireAdmin,
						Value: false,
						Usage: "set execution level to \"Require Administrator\"",
					},
					&cli.StringFlag{
						Name:  flagInfoDescription,
						Usage: "set file description",
					},
					&cli.StringFlag{
						Name:  flagInfoProductName,
						Usage: "set product name",
					},
					&cli.StringFlag{
						Name:  flagInfoCopyright,
						Usage: "set copyright notice",
					},
					&cli.StringFlag{
						Name:  flagInfoFilename,
						Usage: "original filename",
					},
					&cli.StringFlag{
						Name:      flagIconFile,
						Usage:     "icon file (ico, png, ...)",
						TakesFile: true,
					},
				}...),
			},
			{
				Name:   "extract",
				Usage:  "Extract all resources from an executable",
				Action: cmdExtract,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:      flagInput,
						Usage:     "name of the executable file (exe, dll)",
						Required:  true,
						TakesFile: true,
					},
					&cli.StringFlag{
						Name:  flagOutputDir,
						Usage: "name of the output directory",
						Value: defaultOutDir,
					},
					&cli.BoolFlag{
						Name:  flagXMLManifest,
						Usage: "extract the manifest as an xml file (not a json object)",
						Value: false,
					},
				},
			},
			{
				Name:   "replace",
				Usage:  "Replace resources in an executable",
				Action: cmdReplace,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:      flagOutput,
						Usage:     "name of the executable file (exe, dll)",
						Required:  true,
						TakesFile: true,
					},
					&cli.StringFlag{
						Name:  flagInput,
						Usage: "name of the input json file",
						Value: defaultJSONFile,
					},
					&cli.BoolFlag{
						Name:  flagDelete,
						Usage: "delete all resources before adding the new ones",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  flagNoBackup,
						Usage: "don't leave a copy of the original executable",
						Value: false,
					},
				},
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdInit(_ *cli.Context) error {
	err := os.MkdirAll(filepath.Dir(defaultJSONFile), 0666)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(defaultJSONFile, []byte(initJSON), 0666)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(defaultIconFile, initIcon, 0666)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(defaultIcon16File, initIcon16, 0666)
	if err != nil {
		return err
	}

	fmt.Println("Created", defaultJSONFile)

	return nil
}

func cmdMake(ctx *cli.Context) error {
	targets, err := getTargets(ctx)
	if err != nil {
		return err
	}

	rs := &winres.ResourceSet{}
	err = importResources(rs, ctx.String(flagInput))
	if err != nil {
		return err
	}

	err = setVersions(rs, ctx)
	if err != nil {
		return err
	}

	for _, t := range targets {
		err = writeObjectFile(rs, t.name, t.arch)
		if err != nil {
			return err
		}
	}

	return nil
}

func cmdSimply(ctx *cli.Context) error {
	targets, err := getTargets(ctx)
	if err != nil {
		return err
	}

	rs := &winres.ResourceSet{}

	err = simplySeIcon(rs, ctx)
	if err != nil {
		return err
	}

	err = simplySetManifest(rs, ctx)
	if err != nil {
		return err
	}

	simplySetVersionInfo(rs, ctx)

	for _, t := range targets {
		err = writeObjectFile(rs, t.name, t.arch)
		if err != nil {
			return err
		}
	}

	return nil
}

func cmdExtract(ctx *cli.Context) error {
	f, err := os.Open(ctx.String(flagInput))
	if err != nil {
		return err
	}
	defer f.Close()

	rs, err := winres.LoadFromEXE(f)
	if err != nil {
		return err
	}

	f.Close()

	out := ctx.String(flagOutputDir)
	err = os.MkdirAll(out, 0666)
	if err != nil {
		return err
	}

	exportResources(out, rs, !ctx.Bool(flagXMLManifest))

	return nil
}

func cmdReplace(ctx *cli.Context) error {
	exe := ctx.String(flagOutput)

	in, err := os.Open(exe)
	if err != nil {
		return err
	}
	defer in.Close()

	var rs *winres.ResourceSet

	if ctx.Bool(flagDelete) {
		rs = &winres.ResourceSet{}
	} else {
		rs, err = winres.LoadFromEXE(in)
		if err != nil {
			return err
		}
	}

	err = importResources(rs, ctx.String(flagInput))
	if err != nil {
		return err
	}

	out, err := os.Create(exe + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	err = rs.WriteToEXE(out, in)
	if err != nil {
		return err
	}

	err = out.Close()
	if err != nil {
		return err
	}

	if ctx.Bool(flagNoBackup) {
		err = os.Remove(exe)
		if err != nil {
			return err
		}
	} else {
		err = os.Rename(exe, exe+".bak")
		if err != nil {
			return err
		}
	}

	return os.Rename(exe+".tmp", exe)
}

func getTargets(ctx *cli.Context) ([]target, error) {
	var (
		ns   = ctx.Bool(flagNoSuffix)
		arch = strings.Split(ctx.String(flagArch), ",")
		name = ctx.String(flagOutput)
	)

	if ns && len(arch) > 1 {
		return nil, errors.New("cannot use --no-suffix with several targets")
	}

	t := make([]target, len(arch))
	for i := range arch {
		t[i].arch = winres.Arch(strings.TrimSpace(arch[i]))
		switch t[i].arch {
		case winres.ArchAMD64, winres.ArchI386, winres.ArchARM, winres.ArchARM64:
		default:
			return nil, errors.New("unknown architecture: " + arch[i])
		}
		t[i].name = name
		if !ns {
			t[i].name += "_windows_" + string(t[i].arch) + ".syso"
		}
	}

	return t, nil
}

func simplySetManifest(rs *winres.ResourceSet, ctx *cli.Context) error {
	var m *winres.AppManifest

	switch ctx.String(flagManifest) {
	case manifestCLI:
		m = &winres.AppManifest{}

	case manifestGUI:
		m = &winres.AppManifest{DPIAwareness: winres.DPIPerMonitorV2, UseCommonControlsV6: true}

	case manifestNone:

	default:
		return errors.New("invalid manifest type: " + ctx.String(flagManifest))
	}
	if m != nil && ctx.Bool(flagRequireAdmin) {
		m.ExecutionLevel = winres.RequireAdministrator
		rs.SetManifest(*m)
	}

	return nil
}

func simplySetVersionInfo(rs *winres.ResourceSet, ctx *cli.Context) {
	var (
		vi version.Info
		b  bool
	)

	if s := ctx.String(flagInfoDescription); s != "" {
		vi.Set(version.LangDefault, version.FileDescription, s)
		b = true
	}
	if s := ctx.String(flagInfoProductName); s != "" {
		vi.Set(version.LangDefault, version.ProductName, s)
		b = true
	}
	if s := ctx.String(flagInfoFilename); s != "" {
		vi.Set(version.LangDefault, version.OriginalFilename, s)
		b = true
	}
	if s := ctx.String(flagInfoCopyright); s != "" {
		vi.Set(version.LangDefault, version.LegalCopyright, s)
		b = true
	}
	if s := ctx.String(flagFileVersion); s != "" {
		vi.SetFileVersion(s)
		b = true
	}
	if s := ctx.String(flagProductVersion); s != "" {
		vi.SetProductVersion(s)
		b = true
	}

	if b {
		rs.SetVersionInfo(vi)
	}
}

type target struct {
	arch winres.Arch
	name string
}

func setVersions(rs *winres.ResourceSet, ctx *cli.Context) error {
	fileVersion := ctx.String(flagFileVersion)
	prodVersion := ctx.String(flagProductVersion)
	if fileVersion == "" && prodVersion == "" {
		return nil
	}

	var (
		err  error
		done bool
	)
	rs.WalkType(winres.RT_VERSION, func(resID winres.Identifier, langID uint16, data []byte) bool {
		var vi *version.Info
		vi, err = version.FromBytes(data)
		if err != nil {
			return false
		}

		if fileVersion != "" {
			vi.SetFileVersion(fileVersion)
		}

		if prodVersion != "" {
			vi.SetProductVersion(prodVersion)
		}

		done = true

		return true
	})

	if err != nil {
		return err
	}

	if !done {
		vi := version.Info{}
		vi.SetFileVersion(fileVersion)
		vi.SetProductVersion(fileVersion)
		rs.SetVersionInfo(vi)
	}

	return nil
}

func simplySeIcon(rs *winres.ResourceSet, ctx *cli.Context) error {
	name := ctx.String(flagIconFile)
	f, err := os.Open(name)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		log.Println("did not find icon ", name)
		return nil
	}
	defer f.Close()

	var icon *winres.Icon
	if strings.ToLower(filepath.Ext(name)) == ".ico" {
		icon, err = winres.LoadICO(f)
	} else {
		img, _, err := image.Decode(f)
		if err != nil {
			return err
		}
		icon, err = winres.NewIconFromResizedImage(img, nil)
	}

	return rs.SetIcon(winres.ID(1), icon)
}

func writeObjectFile(rs *winres.ResourceSet, name string, arch winres.Arch) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()

	err = rs.WriteObject(f, arch)
	if err != nil {
		return err
	}

	return f.Close()
}

//go:embed icon.png
var initIcon []byte

//go:embed icon16.png
var initIcon16 []byte

var cliManifest = winres.AppManifest{}

var guiManifest = winres.AppManifest{
	Identity:            winres.AssemblyIdentity{},
	DPIAwareness:        winres.DPIPerMonitorV2,
	UseCommonControlsV6: true,
}

// language=json
const initJSON = `{
  "RT_GROUP_ICON": {
    "APP": {
      "0000": [
        "icon.png",
        "icon16.png"
      ]
    }
  },
  "RT_MANIFEST": {
    "#1": {
      "0409": {
        "identity": {
          "name": "",
          "version": ""
        },
        "description": "",
        "minimum-os": "win7",
        "execution-level": "as invoker",
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
        "info": {
          "0409": {
            "Comments": "",
            "CompanyName": "",
            "FileDescription": "",
            "FileVersion": "",
            "InternalName": "",
            "LegalCopyright": "",
            "LegalTrademarks": "",
            "OriginalFilename": "",
            "PrivateBuild": "",
            "ProductName": "",
            "ProductVersion": "",
            "SpecialBuild": ""
          }
        }
      }
    }
  }
}`
