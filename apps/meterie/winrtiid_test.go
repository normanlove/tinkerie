package meterie

import "testing"

func TestPinterfaceIID(t *testing.T) {
	// IAsyncOperation<Boolean>
	if g := pinterfaceIID("pinterface({9fc2b0bb-e446-44e2-aa61-9cab8f636af2};b1)"); g.String() != "{cdb5efb3-5788-509d-9be1-71ccb8a3362a}" {
		t.Errorf("IAsyncOperation<bool>: got %s", g)
	}
	// TypedEventHandler<BluetoothLEAdvertisementWatcher, BluetoothLEAdvertisementReceivedEventArgs>
	sig := typedEventHandlerSig("Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementWatcher", "{a6ac336f-f3d3-4297-8d6c-c81ea6623f40}",
		"Windows.Devices.Bluetooth.Advertisement.BluetoothLEAdvertisementReceivedEventArgs", "{27987ddf-e596-41be-8d43-9e6731d4a913}")
	if g := pinterfaceIID(sig); g.String() != "{90eb4eca-d465-5ea0-a61c-033c8c5ecef2}" {
		t.Errorf("watcher Received handler: got %s", g)
	}
	if g := bleUUID("fff4"); g.String() != "{0000fff4-0000-1000-8000-00805f9b34fb}" {
		t.Errorf("ble uuid %s", g)
	}
}
