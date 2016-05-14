package sfcli

import (
	"fmt"
	"github.com/alecthomas/units"
	"github.com/solidfire/solidfire-docker-driver/sfapi"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"
)

func printStruct(x interface{}) {
	v := reflect.ValueOf(x)
	maxL := 10
	for i := 0; i < v.NumField(); i++ {
		name := v.Type().Field(i).Name
		length := utf8.RuneCountInString(name)
		if length > maxL {
			maxL = length
		}
	}
	l := maxL + 1
	w := strconv.Itoa(l)
	f := "%-" + w + "s %v\n"
	fmt.Println("=======================================")
	fmt.Printf(f, "Property", "Value")
	fmt.Println("=======================================")

	for i := 0; i < v.NumField(); i++ {
		val := v.Field(i).Interface()
		name := v.Type().Field(i).Name
		fmt.Printf(f, name, val)
	}
}

func printVolList(volumes []sfapi.Volume) {
	var provisioned int64
	provisioned = 0

	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	defer tabWriter.Flush()
	fmt.Fprintln(tabWriter, "ID\tName\tStatus\tAccountID\tSize(GiB)\tCreatedAt")
	fmt.Fprintf(tabWriter, "%s\n", "=================================================")
	for _, v := range volumes {
		fmt.Fprintf(tabWriter, "%d\t%s\t%s\t%d\t%d\t%s\n", v.VolumeID, v.Name,
			v.Status, v.AccountID, v.TotalSize/int64(units.GiB), v.CreateTime)
		provisioned += v.TotalSize
	}
	tabWriter.Flush()
	fmt.Println("-------------------------------------------")
	fmt.Println("Total volume count: ", len(volumes))
	fmt.Println("Total GiB provisioned: ", provisioned/int64(units.GiB))
	fmt.Println("-------------------------------------------")
}

func printSnapList(snapshots []sfapi.Snapshot) {
	var provisioned int64
	provisioned = 0
	tabWriter := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)

	defer tabWriter.Flush()
	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\t%s\n", "ID", "NAME", "STATUS", "SIZE(GIB)",
		"PARENT VOLUMEID", "CREATED-AT")
	fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\t%s\n", "==", "====", "======", "=========",
		"===============", "==========")
	for _, s := range snapshots {
		fmt.Fprintf(tabWriter, "%d\t%s\t%s\t%d\t%d\t%s\n", s.SnapshotID, s.Name, s.Status,
			s.TotalSize/int64(units.GiB), s.VolumeID, s.CreateTime)
		provisioned += s.TotalSize
	}
	tabWriter.Flush()
	fmt.Println("-------------------------------------------")
	fmt.Println("Total snapshot count:  ", len(snapshots))
	fmt.Println("Total GiB provisioned: ", provisioned/int64(units.GiB))
	fmt.Println("-------------------------------------------")
}

func confirm() bool {
	var resp string
	_, err := fmt.Scanln(&resp)
	if err != nil {
		fmt.Println("Error scanning input: ", err)
		return false
	}
	resp = strings.ToLower(resp)
	ayes := []string{"y", "yes"}
	for _, a := range ayes {
		if resp == a {
			return true
		}
	}
	return false
}
