NeoCities API Go Wrapper
========================

A NeoCities Go API wrapper.

Provides:

  - Upload, Delete, Info, and List APIs from NeoCities
  - Directory push support

## Usage

To use the API, you will have to create a Site struct
that contains:

  - a key (for uploading/deleting files)
  - a site name (for getting site information)

Example (uploading files):

    package main
    
    import "github.com/vulppine/neocities-go"

    s := neocities.Site{
      Key: [ API key ],
    }

    func main() {
        err := s.Upload("filepath", "", nil)
        // do something with err
    }

## Copyright

Flipp Syder, 2021, MIT License
