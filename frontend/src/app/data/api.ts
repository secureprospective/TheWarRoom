export type PhotoReader = (id: string) => Promise<{ src: string; alt: string }>;

// hostedPhoto is the web app photo: the server photo cache at /v1/headshot/ (inside the tunnel
// path allowlist), so img-src stays self. A missing photo is a 404; the card shows the team.
export const hostedPhoto: PhotoReader = (id) =>
  Promise.resolve(/^[0-9]{1,8}$/.test(id)
    ? { src: `/v1/headshot/350/${id}`, alt: 'Photo' }
    : { src: '', alt: '' });

export function photoReader(): PhotoReader | undefined {
  return typeof window === 'undefined' ? undefined : hostedPhoto;
}

const unavailable = async (): Promise<never> => {
  throw new Error('Local league readings are unavailable in the browser app.');
};
export const targetInjuries = unavailable;
export const targetHandOff = (_id: string) => unavailable();
export const targetWithdrawRoster = (_id: string) => unavailable();
