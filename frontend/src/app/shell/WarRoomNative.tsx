import { hostedWorkflow } from '../commissioner/api';
import { RailLeaguePicker } from './leaguePicker';
import { createPortal } from 'react-dom';
import { readNativeHQ, type NativeHQ } from '../commissioner/hqReading';
import { useWarRoomRead, type WarRoomNative as NativeScope } from './warroomReads';
import WarRoom from './WarRoom';
import { WarRoomEmpty } from './WarRoomHeader';
export default function WarRoomNative(native: NativeScope) {
  const port = native.port ?? hostedWorkflow;
  const reading = useWarRoomRead<NativeHQ>(JSON.stringify([native.league, native.mine]), publish =>
    readNativeHQ(native.league, native.mine, port, publish));
  if (!reading.reading) return <div className={`wr-v6${native.embedded ? ' embedded' : ''}`}>
    {!native.embedded && <RailLeaguePicker />}<WarRoomEmpty reason={reading.error} /></div>;
  const view = <WarRoom snapshot={reading.reading.snapshot}
    native={{ ...native, port, week: reading.reading.week, reading: reading.reading }} />;
  return native.embedded || typeof document === 'undefined' ? view : createPortal(view, document.body);
}
