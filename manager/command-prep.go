package manager

func prepCommand(command string) ([]string, error) {
	if len(command) == 0 {
		return []string{}, nil
	}

	cmd := []string{"sh", "-c", command}
	return cmd, nil
}
