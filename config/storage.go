package config

type Storage interface {
	Subscribe() <-chan Config
	Write(Config) error
}
