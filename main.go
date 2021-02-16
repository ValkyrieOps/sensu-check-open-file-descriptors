package main

import (
        "fmt"
        "bufio"
        "os"
        "os/exec"
        "github.com/sensu-community/sensu-plugin-sdk/sensu"
        "github.com/sensu/sensu-go/types"
        "strconv"
)

// Config represents the check plugin config.
type Config struct {
        sensu.PluginConfig
        User          string
        Warn          int
        Crit          int
}

var (
        plugin = Config{
                PluginConfig: sensu.PluginConfig{
                        Name:     "sensu-ofd-check",
                        Short:    "The Sensu Go Event Open File Descriptors plugin",
                        Keyspace: "sensu.io/plugins/sensu-ofd-check/config",
                },
        }

        options = []*sensu.PluginConfigOption{
                &sensu.PluginConfigOption{
                        Path:      "user",
                        Env:       "CHECK_USER",
                        Argument:  "user",
                        Shorthand: "u",
                        Default:   "sensu",
                        Usage:     "User to query for open file descriptors",
                        Value:     &plugin.User,
                },
                &sensu.PluginConfigOption{
                        Path:      "warn",
                        Env:       "CHECK_WARN",
                        Argument:  "warn",
                        Shorthand: "w",
                        Default:   nil,
                        Usage:     "Warning threshold - count of file descriptors required for warning state",
                        Value:     &plugin.Warn,
                },
                &sensu.PluginConfigOption{
                        Path:      "crit",
                        Env:       "CHECK_CRITICAL",
                        Argument:  "crit",
                        Shorthand: "c",
                        Default:   nil,
                        Usage:     "Critical threshold - count of file descriptors required for critical state",
                        Value:     &plugin.Crit,
                },
        }
)

func main() {
        check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
        check.Execute()
}

// At minimum specify a user to check fd's, note that sudo is required for an accurate count
func checkArgs(event *types.Event) (int, error) {
        if len(plugin.User) == 0 {
                return sensu.CheckStateWarning, fmt.Errorf("--user or CHECK_USER environment variable is required")
        }
        return sensu.CheckStateOK, nil
}


func handleError(err error) {
        if err != nil {
                fmt.Println(err)
                os.Exit(1)
        }
}

func executeCheck(event *types.Event) (int, error) {
// The main logic here is to execute ps -u and loop through the results of within /proc/$process/fd/
        get_pids := exec.Command("ps", "-u", plugin.User, "--no-headers")
        get_pids_num := exec.Command("awk", "{if(NR>1); print $1}")
        get_pids_num.Stdin, _ = get_pids.StdoutPipe()
        stdout, err := get_pids_num.StdoutPipe()

        get_pids_num.Start()
        err = get_pids.Start()
        handleError(err)

        defer get_pids_num.Wait()
        buff := bufio.NewScanner(stdout)

        var pid_count []string

        for buff.Scan() {
                pid_count = append(pid_count, buff.Text())
        }

//	useful for debug
//      fmt.Println("this is pid count", pid_count)

        var fd_count []string

        for _, element := range pid_count {
                get_fds := exec.Command("ls",  "/proc/" +  element + "/fd/")
                count_fds := exec.Command("wc", "-l")

                count_fds.Stdin, _ = get_fds.StdoutPipe()
                stdout, err := count_fds.StdoutPipe()

                count_fds.Start()
		get_fds.Start()
                handleError(err)

                defer count_fds.Wait()
                buff := bufio.NewScanner(stdout)


                for buff.Scan() {
                        fd_count = append(fd_count, buff.Text())
                }
//		useful for debug
//		fmt.Println(fd_count)
}

        sum := 0

        for _, value := range fd_count {
                int_value, _ := strconv.Atoi(value)
                sum += int_value
        }

//		useful for debug
//              fmt.Println(sum)

	if sum >= plugin.Warn && sum < plugin.Crit {
		fmt.Println("WARNING\nOpen File Descriptors for " + plugin.User + ":", sum)
		return sensu.CheckStateWarning, nil
	} else if sum >= plugin.Crit {
		fmt.Println("CRITICAL\nOpen File Descriptors for " + plugin.User + ":", sum)
		return sensu.CheckStateCritical, nil

	} else {
		fmt.Println("OK\nOpen File Descriptors for " + plugin.User + ":", sum)
	        return sensu.CheckStateOK, nil
	}
}
