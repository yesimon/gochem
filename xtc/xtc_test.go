/*
 * untitled.go
 *
 * Copyright 2012 Raul Mera Adasme <rmera_changeforat_chem-dot-helsinki-dot-fi>
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as
 * published by the Free Software Foundation; either version 2.1 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General
 * Public License along with this program.  If not, see
 * <http://www.gnu.org/licenses/>.
 */
/*
 *
 * Gochem is developed at the laboratory for instruction in Swedish, Department of Chemistry,
 * University of Helsinki, Finland.
 *
 *
 */
/***Dedicated to the long life of the Ven. Khenpo Phuntzok Tenzin Rinpoche***/

package xtc

import "fmt"
import "testing"
import "github.com/rmera/gochem"

/*TestXTC reads the frames of the test xtc file using the
 * "interactive" or "low level" functions, i.e. one frame at a time
 * It prints the firs 2 coordinates of each frame and the number of
 * read frames at the end.*/
func TestXTC(Te *testing.T) {
	fmt.Println("First test")
	traj, err := New("../test/test.xtc")
	if err != nil {
		Te.Error(err)
	}
	i := 0
	coords := chem.ZeroVecs(traj.Len())
	for ; ; i++ {
		err := traj.Next(coords)
		if err != nil && err.Error() != "No more frames" {
			Te.Error(err)
			fmt.Println(err)
			break
		} else if err == nil {
			fmt.Println(coords.VecView(2))
		} else {
			fmt.Println("no more!")
			break
		}
	}
	fmt.Println("Over! frames read:", i)
}

/*
//TestFrameXTC reads the frames of the test xtc file from the first to
// the forth frame skipping one frame for each read one. It uses the
// "high level" function. It prints the frames read twince, and the
// coordinates of the forth atom of the last read frame
func TestFrameXTC(Te *testing.T) {
	fmt.Println("Second test!")
	traj,err:=NewXTC("test/test.xtc")
	if err != nil {
		Te.Error(err)
	}
	Coords, read, err := ManyFrames(traj, 0, 5, 1)
	if err != nil {
		Te.Error(err)
	}
	fmt.Println(len(Coords), read, VecView(Coords[read-1],4))
}
*/
func TestFrameXTCConc(Te *testing.T) {
	traj, err := New("../test/test.xtc")
	if err != nil {
		Te.Error(err)
	}
	frames := make([]*chem.VecMatrix, 3, 3)
	for i, _ := range frames {
		frames[i] = chem.ZeroVecs(traj.Len())
	}
	results := make([][]chan *chem.VecMatrix, 0, 0)
	for i := 0; ; i++ {
		results = append(results, make([]chan *chem.VecMatrix, 0, len(frames)))
		coordchans, err := traj.NextConc(frames)
		if err != nil && err.Error() != "No more frames" {
			Te.Error(err)
		} else if err != nil {
			if coordchans == nil {
				break
			}
		}
		for key, channel := range coordchans {
			results[len(results)-1] = append(results[len(results)-1], make(chan *chem.VecMatrix))
			go SecondRow(channel, results[len(results)-1][key], len(results)-1, key)
		}
		res := len(results) - 1
		for frame, k := range results[res] {
			if k == nil {
				fmt.Println(frame)
				continue
			}
			fmt.Println(res, frame, <-k)
		}
	}
}

/*	for framebunch, j := range results {
		if j == nil {
			break
		}
		for frame, k := range j {
			if k == nil {
				fmt.Println(framebunch, frame)
				continue
			}
			fmt.Println(framebunch, frame, <-k)
		}
	}
}
*/
func SecondRow(channelin, channelout chan *chem.VecMatrix, current, other int) {
	if channelin != nil {
		temp := <-channelin
		viej := chem.ZeroVecs(1)
		vector := temp.VecView(2)
		viej.Copy(vector)
		fmt.Println("sending througt", channelin, channelout, viej, current, other)
		channelout <- vector
	} else {
		channelout <- nil
	}
	return
}
