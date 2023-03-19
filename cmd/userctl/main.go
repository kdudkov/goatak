package main

import (
	"bufio"
	"flag"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
	"os"
)

type UserInfo struct {
	User     string `yaml:"user"`
	Callsign string `yaml:"callsign,omitempty"`
	Team     string `yaml:"team,omitempty"`
	Role     string `yaml:"role,omitempty"`
	Password string `yaml:"password"`
}

func read() []*UserInfo {
	dat, err := os.ReadFile("users.yml")
	if err != nil {
		return nil
	}

	users := make([]*UserInfo, 0)
	if err := yaml.Unmarshal(dat, &users); err != nil {
		panic(err.Error())
	}

	return users
}

func write(users []*UserInfo) error {
	f, err := os.Create("users.yml")
	if err != nil {
		return err
	}

	defer f.Close()

	enc := yaml.NewEncoder(f)
	return enc.Encode(users)
}

func main() {
	users := read()

	action := flag.String("a", "", "action")
	passwd := flag.String("password", "", "password")
	user := flag.String("user", "", "user")

	flag.Parse()

	switch *action {
	case "add":
		reader := bufio.NewReader(os.Stdin)

		if *user == "" {
			fmt.Println("need -user")
			return
		}
		for _, u := range users {
			if u.User == *user {
				fmt.Println("user exists")
				return
			}
		}

		pass := *passwd
		if *passwd == "" {
			fmt.Print("password: ")
			p1, _ := reader.ReadString('\n')
			fmt.Print("repeat password: ")
			p2, _ := reader.ReadString('\n')

			if p1 != p2 {
				fmt.Println("\npassword mismatch")
				return
			}
			pass = p1
		}
		bytes, _ := bcrypt.GenerateFromPassword([]byte(pass), 14)

		users = append(users, &UserInfo{User: *user, Password: string(bytes)})
		if err := write(users); err != nil {
			fmt.Println(err.Error())
		}

	case "change":
		reader := bufio.NewReader(os.Stdin)

		if *user == "" {
			fmt.Println("need -user")
			return
		}

		var cu *UserInfo

		for _, u := range users {
			if u.User == *user {
				cu = u
				break
			}
		}

		if cu == nil {
			fmt.Println("user not found")
			return
		}

		pass := *passwd
		if *passwd == "" {
			fmt.Print("password: ")
			p1, _ := reader.ReadString('\n')
			fmt.Print("repeat password: ")
			p2, _ := reader.ReadString('\n')

			if p1 != p2 {
				fmt.Println("\npassword mismatch")
				return
			}
			pass = p1
		}
		bytes, _ := bcrypt.GenerateFromPassword([]byte(pass), 14)
		cu.Password = string(bytes)
		if err := write(users); err != nil {
			fmt.Println(err.Error())
		}

	default:
		for _, user := range users {
			fmt.Printf("%s\t%s\t%s\t%s\n", user.User, user.Callsign, user.Team, user.Role)
		}
	}

}
