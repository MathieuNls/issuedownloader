package pogo

//ReportLinker is an interface that defines
//what a report linker should do
type ReportLinker interface {
	Fetch(string) (Report, error)
	DBName() string
}
