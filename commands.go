package main

import (
	"github.com/samuel/go-zookeeper/zk"
	"sort"
	"strconv"
	"time"
)

func connect() *zk.Conn {
	conn, _, err := zk.Connect([]string{"127.0.0.1:2181"}, time.Second)
	if err != nil {
		panic(err)
	}
	return conn
}

var (
	optWatch bool
)

var cmdExists = &Command{
	Usage: "exists <path> [--watch]",
	Short: "show if a node exists",
	Long: `
Exists checks for a node at the given path and writes "y" or "n" to
stdout according to its presence. If --watch is used, waits for a
change in the presence of the node before exiting.`,
	Run: runWatch,
}

func runWatch(cmd *Command, args []string) {
	if !(len(args) == 1) {
		failUsage(cmd)
	}
	path := args[0]
	conn := connect()
	var events chan zk.Event
	var present bool
	if !optWatch {
		present, _, err := conn.Exists(path)
		must(err)
		
	} else {
		present, _, events, err := conn.ExistsW(path)
		must(err)		
	}
	if present {
		outString("y")
	} else {
		outString("n")
	}
	if events != nil {
		evt := <-events
		must(evt.Err)
	}
}

var cmdStat = &Command {
	Usage: "stat <path>",
	Short: "show node details",
	Long: `
Stat writes to stdout details of the node at the given path.`,
	Run: runStat,
}

func formatTime(millis int64) string {
	t := time.Unix(0, millis*1000000)
	return t.Format(time.RFC3339)
}

func runStat(cmd *Command, args []string) {
	if !(len(args) == 1) {
		failUsage(cmd)
	}
	path := args[0]
	conn := connect()
	_, stat, err := conn.Get(path)
	must(err)
	outString("Czxid:          %d\n", stat.Czxid)
	outString("Mzxid:          %d\n", stat.Mzxid)
	outString("Ctime:          %s\n", formatTime(stat.Ctime))
	outString("Mtime:          %s\n", formatTime(stat.Mtime))
	outString("Version:        %d\n", stat.Version)
	outString("Cversion:       %d\n", stat.Cversion)
	outString("Aversion:       %d\n", stat.Aversion)
	outString("EphemeralOwner: %d\n", stat.EphemeralOwner)
	outString("DataLength:     %d\n", stat.DataLength)
	outString("Pzxid:          %d\n", stat.Pzxid)
}

var cmdGet = &Command{
	Usage: "get <path> [--watch]",
	Short: "show node data",
	Long: `
Get reads the node data at the given path and writes it to stdout. If
--watch is used, waits for a change to the node before exiting.`,
	Run: runGet,
}

func runGet(cmd *Command, args []string) {
	if !(len(args) == 1) {
		failUsage(cmd)
	}
	path := args[0]
	conn := connect()
	if !optWatch {
		data, _, err := conn.Get(path)
		must(err)
		outData(data)
	} else {
		data, _, events, err := conn.GetW(path)
		must(err)
		outData(data)
		evt := <- events
		must(evt.Err)
	}
}

var cmdCreate = &Command {
	Usage: "create <path>",
	Short: "create node with initial data",
	Long: `
Create makes a new node at the given path with the data given by
reading stdin.`,
	Run: runCreate,
}

func runCreate(cmd *Command, args []string) {
	if !(len(args) == 1) {
		failUsage(cmd)
	}
	path := args[0]
	conn := connect()
	data := inData() 
	flags := int32(0)
	acl := zk.WorldACL(zk.PermAll)
	_, err := conn.Create(path, data, flags, acl)
	must(err)
}

var cmdSet = &Command {
	Usage: "set <path> [version]",
	Short: "write node data",
	Long: `
Set updates the node at the given path with the data given by reading
stdin. If a version is given, submits that version with the write
request for verification, otherwise reads the current version before
attempting a write.`,
	Run: runSet,
}

func runSet(cmd *Command, args []string) {
	if !(len(args) == 1 || len(args) == 2) {
		failUsage(cmd)
	}
	path := args[0]
	readVersion := len(args) == 1
	conn := connect()
	data := inData()
	var version int32
	if readVersion {
		_, stat, err := conn.Get(path)
		must(err)
		version = stat.Version
	} else {
		versionParsed, err := strconv.Atoi(args[1])
		must(err)
		version = int32(versionParsed)
	}
	_, err := conn.Set(path, data, version)
	must(err)
}

var cmdDelete = &Command {
	Usage: "delete <path> [version]",
	Short: "delete node",
	Long: `
Delete removes the node at the given path. If a version is given,
submits that version with the write request for verification,
otherwise reads the current version before attempting a write.`,
	Run: runDelete,
}

func runDelete(cmd *Command, args []string) {
	if !(len(args) == 1 || len(args) == 2) {
		failUsage(cmd)
	}
	path := args[0]
	readVersion := len(args) == 1
	conn := connect()
	var version int32
	if readVersion {
		_, stat, err := conn.Get(path)
		must(err)
		version = stat.Version
	} else {
		versionParsed, err := strconv.Atoi(args[1])
		must(err)
		version = int32(versionParsed)
	}
	err := conn.Delete(path, version)
	must(err)
}

var cmdChildren = &Command {
	Usage: "children <path> [--watch]",
	Short: "list children of a node",
	Long: `
Children lists the names of the children of the node at the given
path, one name per line. If --watch is used, waits for a change in the
names of given node's children before returning.`,
	Run: runChildren,
}

func runChildren(cmd *Command, args []string) {
	if !(len(args) == 1) {
		failUsage(cmd)
	}
	path := args[0]
	conn := connect()
	if !optWatch {
		children, _, err := conn.Children(path)
		must(err)
		sort.Strings(children)
		for _, child := range children {
			outString("%s\n", child)
		}
	} else {
		children, _, events, err := conn.ChildrenW(path)
		must(err)
		sort.Strings(children)
		for _, child := range children {
			outString("%s\n", child)
		}
		evt := <- events
		must(evt.Err)
	}
}

func init() {
	cmdExists.Flag.BoolVarP(&optWatch, "watch", "w", false, "watch for a change to node presence before returning")
	cmdGet.Flag.BoolVarP(&optWatch, "watch", "w", false, "watch for a change to node state before returning")
	cmdChildren.Flag.BoolVarP(&optWatch, "watch", "w", false, "watch for a change to node children names before returning")
}