// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

// Package main is the program to fixup cgo generated types
package main

import (
	"go/format"
	"io"
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
)

func main() {
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	b = removeAbsolutePath(b, runtime.GOOS)

	int8variableNames := []string{
		"Buf",
		"Cgroup",
		"Cgroup_name",
		"LocalAddr",
		"LocalAddress",
		"Probe_id",
		"RemoteAddr",
		"RemoteAddress",
		"Request_fragment",
		"Topic_name",
		"Trigger_comm",
		"Victim_comm",
	}

	// Convert []int8 to []byte in multiple generated fields from the kernel, to simplify
	// conversion to string; see golang.org/issue/20753
	convertInt8ArrayToByteArrayRegex := regexp.MustCompile(`(` + strings.Join(int8variableNames, "|") + `)(\s+)\[(\d+)\]u?int8`)
	b = convertInt8ArrayToByteArrayRegex.ReplaceAll(b, []byte("$1$2[$3]byte"))

	// Convert generated pointers to CGo structs to uintptr
	convertPointerToUint64Regex := regexp.MustCompile(`\*_Ctype_struct_(\w+)`)
	b = convertPointerToUint64Regex.ReplaceAll(b, []byte("uintptr"))

	b, err = format.Source(b)
	if err != nil {
		log.Fatal(err)
	}

	os.Stdout.Write(b)
}

// removeAbsolutePath removes the absolute file path that is automatically output by cgo -godefs
// and replaces it with only the filename
func removeAbsolutePath(b []byte, platform string) []byte {
	var removeAbsolutePathRegex *regexp.Regexp
	switch platform {
	case "linux":
		removeAbsolutePathRegex = regexp.MustCompile(`(// cgo -godefs .+) /.+/([^/]+)$`)
	case "windows":
		removeAbsolutePathRegex = regexp.MustCompile(`(// cgo.exe -godefs .+) .:\\.+\\([^\\]+)$`)
	default:
		log.Fatal("unsupported platform")
	}

	return removeAbsolutePathRegex.ReplaceAll(b, []byte("$1 $2"))
}
