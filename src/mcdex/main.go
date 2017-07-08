// ***************************************************************************
//
//  Copyright 2017 David (Dizzy) Smith, dizzyd@dizzyd.com
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
// ***************************************************************************

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var version string

type command struct {
	Fn        func() error
	Desc      string
	ArgsCount int
	Args      string
}

var gCommands = map[string]command{
	"createPack": command{
		Fn:        cmdCreatePack,
		Desc:      "Create a new mod pack",
		ArgsCount: 3,
		Args:      "<directory> <minecraft version> <forge version>",
	},
	"installPack": command{
		Fn:        cmdInstallPack,
		Desc:      "Install a mod pack",
		ArgsCount: 1,
		Args:      "<directory> [<url>]",
	},
	"info": command{
		Fn:        cmdInfo,
		Desc:      "Show runtime info",
		ArgsCount: 0,
	},
	"registerMod": command{
		Fn:        cmdRegisterMod,
		Desc:      "Register a CurseForge mod with an existing pack",
		ArgsCount: 2,
		Args:      "<directory> <url> [<name>]",
	},
	"registerClientMod": command{
		Fn:        cmdRegisterClientMod,
		Desc:      "Register a client-side only CurseForge mod with an existing pack",
		ArgsCount: 2,
		Args:      "<directory> <url> [<name>]",
	},
	"installServer": command{
		Fn:        cmdInstallServer,
		Desc:      "Install a Minecraft server using an existing pack",
		ArgsCount: 1,
		Args:      "<directory>",
	},
}

func cmdCreatePack() error {
	dir := flag.Arg(1)
	minecraftVsn := flag.Arg(2)
	forgeVsn := flag.Arg(3)

	// Create a new pack directory
	cp, err := NewModPack(dir, false)
	if err != nil {
		return err
	}

	// Create the manifest for this new pack
	err = cp.createManifest(dir, minecraftVsn, forgeVsn)
	if err != nil {
		return err
	}

	// Create the launcher profile (and install forge if necessary)
	err = cp.createLauncherProfile()
	if err != nil {
		return err
	}

	return nil
}

func cmdInstallPack() error {
	dir := flag.Arg(1)
	url := flag.Arg(2)

	// Only require a manifest if we're not installing from a URL
	requireManifest := (url == "")

	cp, err := NewModPack(dir, requireManifest)
	if err != nil {
		return err
	}

	if url != "" {
		// Download the pack
		err = cp.download(url)
		if err != nil {
			return err
		}

		// Process manifest
		err = cp.processManifest()
		if err != nil {
			return err
		}

		// Install overrides from the modpack; this is a bit of a misnomer since
		// under usual circumstances there's are no mods in the modpack file that
		// will be also be downloaded
		err = cp.installOverrides()
		if err != nil {
			return err
		}
	}

	// Create launcher profile
	err = cp.createLauncherProfile()
	if err != nil {
		return err
	}

	// Install mods (include client-side only mods)
	err = cp.installMods(true)
	if err != nil {
		return err
	}

	return nil
}

func cmdInfo() error {
	fmt.Printf("Version: %+s\n", version)
	fmt.Printf("Env: %+v\n", env())
	return nil
}

func cmdRegisterMod() error {
	return _registerMod(false)
}

func cmdRegisterClientMod() error {
	return _registerMod(true)
}

func _registerMod(clientOnly bool) error {
	dir := flag.Arg(1)
	url := flag.Arg(2)
	name := flag.Arg(3)

	if !strings.Contains(url, "minecraft.curseforge.com") && name == "" {
		return fmt.Errorf("Insufficient arguments")
	}

	cp, err := NewModPack(dir, true)
	if err != nil {
		return err
	}

	err = cp.registerMod(url, name, clientOnly)
	if err != nil {
		return err
	}

	return nil
}

func cmdInstallServer() error {
	dir := flag.Arg(1)

	// Open the pack; we require the manifest and any
	// config files to already be present
	cp, err := NewModPack(dir, true)
	if err != nil {
		return err
	}

	// Install the server jar, Forge and dependencies
	err = cp.installServer()
	if err != nil {
		return err
	}

	// Make sure all mods are installed
	err = cp.installMods(false)
	if err != nil {
		return err
	}

	return nil
	// Setup the command-line
	// java -jar <forge.jar>
}

func console(f string, args ...interface{}) {
	fmt.Printf(f, args...)
}

func usage() {
	console("usage: mcdex [<options>] <command> [<args>]\n")
	console(" commands:\n")
	for id, cmd := range gCommands {
		console(" - %s: %s\n", id, cmd.Desc)
	}
}

func usageCmd(name string, cmd command) {
	console("usage: mcdex %s %s\n", name, cmd.Args)
}

func main() {
	// Process command-line args
	flag.Parse()
	if !flag.Parsed() || flag.NArg() < 1 {
		usage()
		os.Exit(-1)
	}

	// Initialize our environment
	err := initEnv()
	if err != nil {
		log.Fatalf("Failed to initialize: %s\n", err)
	}

	commandName := flag.Arg(0)
	command, exists := gCommands[commandName]
	if !exists {
		console("ERROR: unknown command '%s'\n", commandName)
		usage()
		os.Exit(-1)
	}

	// Check that the required number of arguments is present
	if flag.NArg() < command.ArgsCount+1 {
		console("ERROR: insufficient arguments for %s\n", commandName)
		console("usage: mcdex %s %s\n", commandName, command.Args)
		os.Exit(-1)
	}

	err = command.Fn()
	if err != nil {
		log.Fatalf("%+v\n", err)
	}
}

//mcdex update - download latest mcdex.sqlite
//mcdex forge.install <name> [<vsn>]
//mcdex forge.list

//mcdex init <name> <vsn> <desc>
//mcdex install <modname> [<vsn>]
