// +build windows

package manager

func prepCommand(command string) ([]string, error) {
	switch {
	case len(command) == 0:
		return []string{}, nil
	case len(strings.Fields(command)) > 1:
		return cmd, fmt.Errorf("only single commands supported on windows")
	}
	return []string{command}, nil
}
