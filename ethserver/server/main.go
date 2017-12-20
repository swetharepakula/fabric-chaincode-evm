package main

func Main() {
	ethService := ethserver.NewEthService()
	server := ethserver.NewEthServer(ethService)

	server.Start(5000)
}
