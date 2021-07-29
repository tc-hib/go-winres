package main

import (
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	flagAuthenticode = "authenticode"

	authenticodeIgnore = "ignore"
	authenticodeRemove = "remove"

	gitTag = "git-tag"
)

//go:generate go-winres make --product-version=git-tag --file-version=git-tag

func main() {
	versionFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  flagProductVersion,
			Usage: `set product version (special value: "` + gitTag + `")`,
		},
		&cli.StringFlag{
			Name:  flagFileVersion,
			Usage: `set file version (special value: "` + gitTag + `")`,
		},
	}

	commonMakeFlags := append([]cli.Flag{
		&cli.StringFlag{
			Name:    flagArch,
			Usage:   "comma separated list of target architectures",
			Value:   defaultArch,
			EnvVars: []string{"GOARCH"},
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
	}, versionFlags...)

	app := cli.App{
		Name:  "go-winres",
		Usage: "A tool for embedding resources in Windows executables",
		Commands: []*cli.Command{
			{
				Name:      "init",
				Usage:     "Create an initial ./winres/winres.json",
				Action:    cmdInit,
				ArgsUsage: " ",
			},
			{
				Name:      "make",
				Usage:     "Make syso files for the \"go build\" command",
				Action:    cmdMake,
				ArgsUsage: " ",
				Flags: append([]cli.Flag{
					&cli.StringFlag{
						Name:      flagInput,
						Usage:     "name of the input json file",
						Value:     defaultJSONFile,
						TakesFile: true,
					},
				},
					commonMakeFlags...),
			},
			{
				Name:      "simply",
				Usage:     "Make syso files for the \"go build\" command (simplified)",
				Action:    cmdSimply,
				ArgsUsage: " ",
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
						Value:     defaultIconFile,
						TakesFile: true,
					},
				}...),
			},
			{
				Name:      "extract",
				Usage:     "Extract all resources from an executable",
				Action:    cmdExtract,
				ArgsUsage: "source_file.exe",
				Flags: []cli.Flag{
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
				Name:      "patch",
				Usage:     "Replace resources in an executable file (exe, dll)",
				Action:    cmdPatch,
				ArgsUsage: "target_file.exe",
				Flags: append([]cli.Flag{
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
					&cli.StringFlag{
						Name:  flagAuthenticode,
						Usage: `specify what to do with signed executables: "ignore" or "remove" signature`,
						Value: "",
					},
				}, versionFlags...),
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

	err = simplySetIcon(rs, ctx)
	if err != nil {
		return err
	}

	err = simplySetManifest(rs, ctx)
	if err != nil {
		return err
	}

	err = simplySetVersionInfo(rs, ctx)
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

func cmdExtract(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		cli.ShowSubcommandHelpAndExit(ctx, 1)
	}

	f, err := os.Open(ctx.Args().Get(0))
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

func cmdPatch(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		cli.ShowSubcommandHelpAndExit(ctx, 1)
	}

	exe := ctx.Args().Get(0)

	in, err := os.Open(exe)
	if err != nil {
		return err
	}
	defer in.Close()

	signed, err := winres.IsSignedEXE(in)
	if err != nil {
		return err
	}
	if signed {
		switch ctx.String(flagAuthenticode) {
		case authenticodeRemove, authenticodeIgnore:
		default:
			return fmt.Errorf(`Cannot patch a signed file.

You might want to use --%[1]s=%[2]s or --%[1]s=%[3]s to either remove or ignore the code signature.`, flagAuthenticode, authenticodeRemove, authenticodeIgnore)
		}
	}

	var rs *winres.ResourceSet

	if ctx.Bool(flagDelete) {
		rs = &winres.ResourceSet{}
	} else {
		rs, err = winres.LoadFromEXE(in)
		if err != nil && err != winres.ErrNoResources {
			return err
		}
	}

	err = importResources(rs, ctx.String(flagInput))
	if err != nil {
		return err
	}

	err = setVersions(rs, ctx)
	if err != nil {
		return err
	}

	out, err := os.Create(exe + ".tmp")
	if err != nil {
		return err
	}
	defer out.Close()

	switch ctx.String(flagAuthenticode) {
	case authenticodeRemove:
		err = rs.WriteToEXE(out, in, winres.WithAuthenticode(winres.RemoveSignature))
	case authenticodeIgnore:
		err = rs.WriteToEXE(out, in, winres.WithAuthenticode(winres.IgnoreSignature))
	default:
		err = rs.WriteToEXE(out, in)
	}
	if err != nil {
		return err
	}

	in.Close()
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
	if ctx.Bool(flagRequireAdmin) {
		if m == nil {
			m = &winres.AppManifest{}
		}
		m.ExecutionLevel = winres.RequireAdministrator
	}

	if m != nil {
		rs.SetManifest(*m)
	}

	return nil
}

func simplySetVersionInfo(rs *winres.ResourceSet, ctx *cli.Context) error {
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

	fileVersion, prodVersion, err := getInputVersions(ctx)
	if err != nil {
		return err
	}
	if fileVersion != "" {
		vi.SetFileVersion(fileVersion)
		b = true
	}
	if prodVersion != "" {
		vi.SetProductVersion(prodVersion)
		b = true
	}

	if b {
		rs.SetVersionInfo(vi)
	}

	return nil
}

type target struct {
	arch winres.Arch
	name string
}

func getInputVersions(ctx *cli.Context) (fileVersion string, prodVersion string, err error) {
	fileVersion = ctx.String(flagFileVersion)
	prodVersion = ctx.String(flagProductVersion)
	if fileVersion != gitTag && prodVersion != gitTag {
		return
	}
	tag, err := getGitTag()
	if err != nil {
		fileVersion = ""
		prodVersion = ""
		return
	}
	if fileVersion == gitTag {
		fileVersion = tag
	}
	if prodVersion == gitTag {
		prodVersion = tag
	}
	return
}

func setVersions(rs *winres.ResourceSet, ctx *cli.Context) error {
	fileVersion, prodVersion, err := getInputVersions(ctx)
	if err != nil {
		return err
	}

	done := false
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

		rs.SetVersionInfo(*vi)

		done = true

		return true
	})

	if err != nil {
		return err
	}

	if !done {
		vi := version.Info{}
		if fileVersion != "" {
			vi.SetFileVersion(fileVersion)
		}
		if prodVersion != "" {
			vi.SetProductVersion(prodVersion)
		}
		rs.SetVersionInfo(vi)
	}

	return nil
}

func getGitTag() (string, error) {
	w := strings.Builder{}
	cmd := exec.Command("git", "describe", "--tags")
	cmd.Stdout = &w
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(w.String()), nil
}

func simplySetIcon(rs *winres.ResourceSet, ctx *cli.Context) error {
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
		if err != nil {
			return err
		}
	} else {
		img, _, err := image.Decode(f)
		if err != nil {
			return err
		}
		icon, err = winres.NewIconFromResizedImage(img, nil)
		if err != nil {
			return err
		}
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
      "0000": {
        "fixed": {
          "file_version": "0.0.0.0",
          "product_version": "0.0.0.0"
        },
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
