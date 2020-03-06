package bot

import "strconv"

func parseID(id string) (uint64, error) {
	return strconv.ParseUint(id, 10, 64)
}
