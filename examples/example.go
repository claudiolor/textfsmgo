package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/claudiolor/textfsmgo/pkg/textfsmgo"
	"github.com/claudiolor/textfsmgo/pkg/utils"
)

type NetworkIf struct {
	ifname    string
	macaddr   string
	addresses []string
	mtu       int
	state     string
}

func showError(err error, ecode int) {
	fmt.Println(err.Error())
	os.Exit(ecode)
}

func useAddressList(l []NetworkIf) {
	fmt.Println(l)
}

func main() {
	in_file := "./data/ip_cmd.raw"
	tmpl_file := "./data/ip_cmd.textfsm"
	parser, err := textfsmgo.NewTextFSMParser(tmpl_file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	input_str, err := os.ReadFile(in_file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	res, err := parser.ParseTextToDicts(string(input_str))
	if err != nil {
		showError(err, 1)
	}

	// Result to struct
	addresses_list := []NetworkIf{}
	for _, entry := range res {
		mtu, _ := strconv.Atoi(entry["mtu"].(string))
		addresses_list = append(
			addresses_list,
			NetworkIf{
				ifname:    entry["ifname"].(string),
				macaddr:   entry["macaddr"].(string),
				addresses: entry["addresses"].([]string),
				state:     entry["state"].(string),
				mtu:       mtu,
			},
		)
	}
	useAddressList(addresses_list)

	// Convert to json
	if jsonres, err := utils.ConvertResToJson(&res, true); err != nil {
		showError(err, 1)
	} else {
		fmt.Println(string(jsonres))
	}
}
