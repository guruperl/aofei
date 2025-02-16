package main

import (
	"bufio"
	"context"
	"database/sql"
	"os"
	"regexp"
	"strconv"
)

func doChannel(ctx context.Context, db *sql.DB) error {
	f, err := os.Open("channel.proto")
	if err != nil {
		return err
	}
	defer f.Close()

	ref := make(map[int]int)
	re := regexp.MustCompile(`^\s+(IAB(\d+)(_\d+)?)\s=\s(\d+);\s+\/\/\s(.*)$`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		arr := re.FindStringSubmatch(line)
		if len(arr) != 6 {
			continue
		}
		parent := 0
		level := 1
		channelName := arr[1]
		fullName := arr[5]
		channelID, err := strconv.Atoi(arr[4])
		if err != nil {
			return err
		}
		id, err := strconv.Atoi(arr[2])
		if err != nil {
			panic(err)
		}
		if arr[3] == "" {
			ref[id] = channelID
		} else {
			parent = ref[id]
			level = 2
		}
		_, err = db.ExecContext(ctx, `
INSERT INTO def_channel (channel_id,channel_name,level,parent,full_name)
VALUES (?,?,?,?,?)`, channelID, channelName, level, parent, fullName)
		if err != nil {
			return err
		}
	}
	return scanner.Err()
}
