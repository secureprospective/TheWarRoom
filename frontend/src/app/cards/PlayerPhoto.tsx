import { useEffect, useState } from 'react';

// Go's verdict on a player's photo (W1): an empty src means show the fallback, no request.
type PhotoView = { src: string; alt: string };
type PhotoReader = (id: string) => Promise<PhotoView>;

type GoWindow = { go?: { main?: { App?: { TargetPhoto?: PhotoReader } } } };

function goReader(): PhotoReader | undefined {
  if (typeof window === 'undefined') return undefined;
  return (window as unknown as GoWindow).go?.main?.App?.TargetPhoto;
}

export function usePhoto(id: string, read = goReader()): PhotoView | undefined {
  const [view, setView] = useState<PhotoView>();
  useEffect(() => {
    let live = true;
    setView(undefined);
    read?.(id).then((v) => live && setView(v), () => live && setView(undefined));
    return () => {
      live = false;
    };
  }, [id, read]);
  return view;
}

// The box is reserved at its size, so a photo arriving late never shifts the layout; without a
// photo it shows the NFL team, never a broken image.
type PhotoProps = { view?: PhotoView; team?: string; size?: 'sm' | 'md' };

export function PlayerPhoto({ view, team, size = 'md' }: PhotoProps) {
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [view?.src]);
  const show = view?.src && !failed;
  return (
    <span className={`player-photo player-photo-${size}`} aria-hidden={show ? undefined : true}>
      {show ? (
        <img src={view.src} alt={view.alt} loading="lazy" decoding="async"
          onError={() => setFailed(true)} />
      ) : (
        <span className="player-photo-fallback">{team ?? ''}</span>
      )}
    </span>
  );
}
