// Copyright 2017 someonegg. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utility

import "time"

func NowInMS() int64 {
	now := time.Now()
	return int64(now.Unix()*1000) + int64(now.Nanosecond()/1e6)
}
