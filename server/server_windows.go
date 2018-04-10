package server

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/davecgh/go-spew/spew"
	"github.com/sethdmoore/serial-hotkey/hotkeys"
	"github.com/sethdmoore/serial-hotkey/serial"
	"github.com/sethdmoore/serial-hotkey/util"
	"github.com/sethdmoore/serial-hotkey/windows"
)

func Start() {

	serialPort := "COM1"

	port, err := serial.Connect(serialPort)
	if err != nil {
		log.Fatalf("Could not connect to %s, %v", serialPort, err)
	}
	defer port.Close()

	wincalls := windows.Get()

	var msg windows.MSG

	fmt.Println("running")

	keys := hotkeys.Keys

	var keystate windows.KeyState

	for {
		r1, _, err := wincalls.GetMSG.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 1)
		if r1 == 0 {
			log.Printf("Error parsing windows message: %v, %v", r1, err)
		}

		// Registered id is in the WPARAM field:
		if id := msg.WPARAM; id != 0 {
			keystate.KeyCode = keys[id].KeyCode

			linuxkey, err := util.WinKeyToLinux(keystate.KeyCode)
			if err != nil {
				log.Printf("Warning: problem mapping Windows key %d to Linux: .. %v", keystate.KeyCode, err)
				continue
			}

			if keys[id].Modifiers&windows.ModNoRepeat != 0 {
				log.Printf("Key %d being held...\n", keystate.KeyCode)
				_, err := port.Write([]byte(fmt.Sprintf("down:%d\n", linuxkey)))
				if err != nil {
					log.Fatalf("port.Write: %v", err)
				}
			inner:
				for {
					time.Sleep(10 * time.Millisecond)
					r1, _, _ := wincalls.KeyState.Call(uintptr(keystate.KeyCode))
					if r1 == 0 {
						log.Printf("Key %d released!\n", keystate.KeyCode)

						_, err := port.Write([]byte(fmt.Sprintf("up:%d\n", linuxkey)))
						if err != nil {
							log.Fatalf("port.Write: %v", err)
						}
						break inner
					}

				}

			} else {
				fmt.Println("Hotkey pressed:", keys[id])
				_, err := port.Write([]byte(fmt.Sprintf("press:%d\n", linuxkey)))
				if err != nil {
					log.Fatalf("port.Write: %v", err)
				}
			}

			if id == 3 { // CTRL+ALT+X = Exit
				fmt.Println("CTRL+ALT+X pressed, goodbye...")
				return
			}
		} else {
			spew.Dump(msg)
		}

		// Not sure if this section is required
		// MSDN documentation is shy on this
		wincalls.TranslateMSG.Call(uintptr(unsafe.Pointer(&msg)))
		wincalls.DispatchMSG.Call(uintptr(unsafe.Pointer(&msg)))

		time.Sleep(time.Millisecond * 50)
	}
}
