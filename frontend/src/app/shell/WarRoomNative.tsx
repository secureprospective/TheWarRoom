import { RailLeaguePicker } from './leaguePicker';
import { createPortal } from 'react-dom';
import { readNativeHQ } from '../commissioner/hqReading';
import { useWarRoomRead, type WarRoomNative as NativeScope } from './warroomReads';
import WarRoom from './WarRoom';
import { WarRoomEmpty } from './WarRoomHeader';
export default function WarRoomNative(native: NativeScope) {
  const reading = useWarRoomRead(JSON.stringify([native.league, native.mine]), () =>
    readNativeHQ(native.league, native.mine, native.port));
  if (!reading.reading) return <div className="wr-v6"><RailLeaguePicker /><WarRoomEmpty reason={reading.error} /></div>;
  const view = <WarRoom snapshot={reading.reading.snapshot} native={{ ...native, week: reading.reading.week }}
    />;
  return typeof document === 'undefined' ? view : createPortal(view, document.body);
}
