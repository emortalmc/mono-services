package utils

import "strings"

// ParseFleetFromPodName parses the fleet name from a server id
// The format of a pod name is <fleetName>-<random>-<random>
// fleetName may contain additional hyphens
func ParseFleetFromPodName(podName string) string {
	split := strings.Split(podName, "-")

	return strings.Join(split[:len(split)-2], "-")
}
