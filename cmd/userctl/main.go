package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"github.com/kdudkov/goatak/internal/model"
)

func read(fn string) []*model.UserInfo {
	dat, err := os.ReadFile(fn)
	if err != nil {
		return nil
	}

	users := make([]*model.UserInfo, 0)
	if err := yaml.Unmarshal(dat, &users); err != nil {
		panic(err.Error())
	}

	return users
}

func write(fn string, users []*model.UserInfo) error {
	f, err := os.Create(fn)
	if err != nil {
		return err
	}

	defer f.Close()

	enc := yaml.NewEncoder(f)
	return enc.Encode(users)
}

func main() {
	file := flag.String("file", "users.yml", "file")
	user := flag.String("user", "", "user")
	passwd := flag.String("password", "", "password")
	scope := flag.String("scope", "", "scope")

	users := read(*file)

	flag.Parse()

	if *user == "" {
		for _, user := range users {
			fmt.Printf("%s\t%s\t%s\t%s\n", user.User, user.Callsign, user.Team, user.Role)
		}
		return
	}

	pass := *passwd
	if pass == "" {
		reader := bufio.NewReader(os.Stdin)
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

	bpass, _ := bcrypt.GenerateFromPassword([]byte(pass), 14)

	var found bool

	for _, u := range users {
		if u.User == *user {
			found = true
			u.Password = string(bpass)
			if *scope != "" {
				u.Scope = *scope
			}
			break
		}
	}

	if !found {
		users = append(users, &model.UserInfo{User: *user, Password: string(bpass), Scope: *scope})
	}

	if err := write(*file, users); err != nil {
		fmt.Println(err.Error())
	}
}
