# go-winres

A simple command line tool for embedding usual resources in Windows executables built with Go:

- A manifest
- An application icon
- Version information (the Details tab in file properties)
- Other icons and cursors

You might want to directly use winres as a library too: [github.com/tc-hib/winres](https://github.com/tc-hib/winres)

## Installation

To install the go-winres command, run:

```shell
go install github.com/tc-hib/go-winres@latest
```

## Usage

Please type `go-winres help` to get a list of commands and options.

Typical usage would be:

* Run `go-winres init` to create a `winres` directory
* Modify the contents of `winres.json`
* Before `go build`, run `go-winres make`

`go-winres make` creates files named `rsrc_windows_*.syso` that `go build` automatically embeds in the executable.

The suffix `_windows_amd64` is very important.
Thanks to it, `go build` knows it should not include that object in a Linux or 386 build.

### Automatic version from git

The `--file-version` and `--product-version` flags can take a special value: `git-tag`.
This will retrieve the current tag with `git describe --tags` and add it to the file properties of the executable.

### Using `go generate`

You can use a `//go:generate` comment as well:

```
//go:generate go-winres make --product-version=git-tag
```

### Subcommands

There are other subcommands:

* `go-winres simply` is a simpler `make` that does not rely on a json file.
* `go-winres extract` extracts resources from an `exe` file or a `dll`.
* `go-winres patch` replaces resources directly in an `exe` file or a `dll`.
  For example, to enhance a 7z self extracting archive, you may change its icon,
  and add a manifest to make it look better on high DPI screens.

## JSON format

The JSON file follows this hierarchy:

* Resource type (e.g. `"RT_GROUP_ICON"` or `"#42"` or `"MY_TYPE"`)
    * Resource name (e.g. `"MY_ICON"` or `"#1"`)
        * Language ID (e.g. `"0409"` for en-US)
            * Actual resource: a filename or a json structure

Standard resource types can be found [there](https://docs.microsoft.com/en-us/windows/win32/menurc/resource-types). But
please never use `RT_ICON` or `RT_CURSOR`. Use `RT_GROUP_ICON` and `RT_GROUP_CURSOR` instead.

### Icon JSON

```json
{
  "RT_GROUP_ICON": {
    "APP": {
      "0000": [
        "icon_64.png",
        "icon_48.png",
        "icon_32.png",
        "icon_16.png"
      ]
    },
    "OTHER": {
      "0000": "icon.png"
    },
    "#42": {
      "0409": "icon_EN.ico",
      "040C": "icon_FR.ico"
    }
  }
}
```

This example contains 3 icons:

* `"APP"`
* `"OTHER"`
* `42`

Windows Explorer will display `"APP"` because it is the first one. Icons are sorted by name in case sensitive ascending
order, then by ID.

`42` is an ID, not a name, this is why it comes last.

* `"APP"` is made of 4 png files.
* `"OTHER"` will be generated from one png file. It will be resized to 256x256, 64x64, 48x48, 32x32, and 16x16.
* `42` is a native icon, it probably already contains several images.

Finally, `42` will display a different icon for french users.

* `"0409"` means en-US, which is the default.
* `"040C"` means fr-FR.

You can find other language IDs [there](https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-lcid/).

### Cursor JSON

```json
{
  "RT_GROUP_CURSOR": {
    "ARROW": {
      "0000": [
        {
          "image": "arrow_32.png",
          "x": 28,
          "y": 4
        },
        {
          "image": "arrow_48.png",
          "x": 40,
          "y": 6
        }
      ]
    },
    "MOVE": {
      "0409": "move_EN.cur",
      "040C": "move_FR.cur"
    },
    "#1": {
      "0000": {
        "image": "cross.png",
        "x": 16,
        "y": 16
      }
    }
  }
}
```

This example contains 3 cursors:

* `"ARROW"` contains two images (one for higher DPI). It is a json array.
* `"MOVE"` uses cur files directly. It is different in French. It is a string.
* `1` contains one image. It is an object.

When a cursor is made with a png file, you have to provide the coordinates of the "hot spot", that is, the pixel that
clicks.

### Manifest

The manifest should be defined as resource `1` with language `0409`.

#### As a JSON object

```json
{
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
  }
}
```

All boolean values default to `false`.

It is recommended to omit `identity` if your program is a plain application, not meant to be a side-by-side
dependency.

##### Values for `"execution-level"`:

- `""` (default)
- `"highest"`: elevates to the highest level available to the current user
- `"administrator"`: require the user to be an administrator and elevate to this level

##### Values for `"minimum-os"`:

- `"vista"`
- `"win7"` (default)
- `"win8"`
- `"win8.1"`
- `"win10"`

##### Values for `"dpi-awareness"`:

- `"unaware"`
- `"system"` (default)
- `"per monitor"`
- `"per monitor v2"` (recommended)

#### As an XML file

```json
{
  "RT_MANIFEST": {
    "#1": {
      "0409": "my_manifest.xml"
    }
  }
}
```

### VersionInfo JSON

Here is an example JSON file containing every standard info field, a French translation, and every possible flag.
`"0409"` and `"040C"` are language code identifiers (LCID) for `en-US` and `fr-FR` respectively.

```json
{
  "RT_VERSION": {
    "#1": {
      "0000": {
        "fixed": {
          "file_version": "1.2.3.4",
          "product_version": "1.2.3.42",
          "flags": "Debug,Prerelease,Patched,PrivateBuild,SpecialBuild",
          "timestamp": "2020-12-18T23:00:00+01:00"
        },
        "info": {
          "0409": {
            "Comments": "Comments",
            "CompanyName": "Company",
            "FileDescription": "A description",
            "FileVersion": "1.2.3.4",
            "InternalName": "",
            "LegalCopyright": "© You",
            "LegalTrademarks": "",
            "OriginalFilename": "X.EXE",
            "PrivateBuild": "",
            "ProductName": "Product",
            "ProductVersion": "1.2.3.42 beta",
            "SpecialBuild": ""
          },
          "040C": {
            "Comments": "Commentaire",
            "CompanyName": "Compagnie",
            "FileDescription": "Une description",
            "FileVersion": "1.2.3.4",
            "InternalName": "",
            "LegalCopyright": "© Vous",
            "LegalTrademarks": "",
            "OriginalFilename": "X.EXE",
            "PrivateBuild": "",
            "ProductName": "Produit",
            "ProductVersion": "1.2.3.42 bêta",
            "SpecialBuild": ""
          }
        }
      }
    }
  }
}
```

## Alternatives

This project is similar to [akavel/rsrc](https://www.github.com/akavel/rsrc/)
and [josephspurrier/goversioninfo](https://github.com/josephspurrier/goversioninfo).

Additional features are:

- Multilingual resources
- Multilingual VersionInfo that works in Windows Explorer
- Explicitly named resources, by ID or by string (so you can use them in runtime)
- Extracting resources from `exe` or `dll` files
- Replacing resources in `exe` or `dll` files
- Simplified VersionInfo definition
- Simplified manifest definition
- Support for custom information in VersionInfo
- Making an icon or a cursor from a PNG file
- Embedding custom resources

It might be closer to Microsoft specifications too.

## Limitations

`go-winres` is not a real resource compiler, which means it won't help you embed these UI templates:

- `ACCELERATORS`
- `DIALOGEX`
- `MENUEX`
- `POPUP`

If you ever need them, you can use one of those tools instead:

- `rc.exe` and `cvtres.exe` from Visual Studio
- `windres` from GNU Binary Utilities
- `llvm-rc` and `llvm-cvtres` from LLVM tools

See [Resource Compiler](https://docs.microsoft.com/en-us/windows/win32/menurc/resource-compiler) for more information.

## Thanks

Many thanks to [akavel](https://github.com/akavel) for his help.

This project uses these very helpful libs:

- [nfnt/resize](github.com/nfnt/resize) - pure Go image resizing
- [urfave/cli](github.com/urfave/cli) - makes building a CLI easier
