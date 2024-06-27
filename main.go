/*
Copyright © 2020 Leigh MacDonald <leigh.macdonald@gmail.com>
*/
package main

import (
	"github.com/viciious/mika/cmd"
	//_ "github.com/viciious/mika/store/http"
	_ "github.com/viciious/mika/store/memory"
	_ "github.com/viciious/mika/store/mysql"
	//_ "github.com/viciious/mika/store/postgres"
	_ "github.com/viciious/mika/store/redis"
)

func main() {
	cmd.Execute()
}
