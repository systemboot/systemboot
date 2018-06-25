// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package fit

import (
	"io/ioutil"
	"os"
)

func (n *Node) parseDirectory(dir string) (err error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			c := &Node{}
			c.Name = file.Name()
			c.Depth = n.Depth + 1
			if n.Children == nil {
				n.Children = make(map[string]*Node)
			}
			n.Children[c.Name] = c
			c.parseDirectory(dir + string(os.PathSeparator) + c.Name)
		} else {
			if n.Properties == nil {
				n.Properties = make(map[string][]byte)
			}
			v, err := ioutil.ReadFile(dir + string(os.PathSeparator) +
				file.Name())
			if err == nil {
				n.Properties[file.Name()] = v
			}
		}
	}
	return nil
}

func (t *Tree) ParseKernel() (err error) {
	var n *Node
	if n = t.RootNode; n == nil {
		n = &Node{}
		n.Name = "/"
		n.Depth = 1
		t.RootNode = n
	}
	err = n.parseDirectory("/proc/device-tree")

	return err
}
