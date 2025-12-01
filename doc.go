// Package sonos provides an implementation of the Sonos UPnP API.
//
// This package allows discovery, control, and event subscription for Sonos devices on the local network.
//
// Basic Usage:
//
//	package main
//
//	import (
//		"context"
//		"log"
//		"time"
//
//		"github.com/caglar10ur/sonos"
//	)
//
//	func main() {
//		s, err := sonos.NewSonos()
//		if err != nil {
//			log.Fatal(err)
//		}
//		defer s.Close()
//
//		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//		defer cancel()
//
//		// Discover devices
//		s.Search(ctx, func(s *sonos.Sonos, zp *sonos.ZonePlayer) {
//			log.Printf("Found device: %s (%s)", zp.RoomName(), zp.IPAddress())
//
//			// Control the device (e.g., GetVolume)
//			vol, err := zp.GetVolume()
//			if err != nil {
//				log.Printf("Error getting volume: %v", err)
//				return
//			}
//			log.Printf("Current volume: %d", vol)
//		})
//
//		<-ctx.Done()
//	}
package sonos
