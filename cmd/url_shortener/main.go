package url_shortener

import (
	"fmt"
	"urlShortener/internal/config"
)

func main() {
	//TODO: init config
	cfg := config.MustLoad()

	fmt.Println(cfg)
	//TODO: init log

	//TODO: init storage

	//TODO: init router

	//TODO: run server
}
