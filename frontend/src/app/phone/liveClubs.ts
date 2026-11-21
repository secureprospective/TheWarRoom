/* Club colours for the live field, with the mock v1 colour math: every team colour on screen is derived from
   the club's two real colours and checked for contrast against what it sits on. */
export type Club = { ab: string; name: string; pri: string; sec: string };
type RGB = [number, number, number];

/* city name, primary, secondary: the clubs' published colours */
const CLUBS: Record<string, [string, string, string]> = {
  ARI: ['ARIZONA', '#97233F', '#FFB612'], ATL: ['ATLANTA', '#A71930', '#A5ACAF'],
  BAL: ['BALTIMORE', '#241773', '#9E7C0C'], BUF: ['BUFFALO', '#00338D', '#C60C30'],
  CAR: ['CAROLINA', '#0085CA', '#BFC0BF'], CHI: ['CHICAGO', '#0B162A', '#C83803'],
  CIN: ['CINCINNATI', '#FB4F14', '#000000'], CLE: ['CLEVELAND', '#311D00', '#FF3C00'],
  DAL: ['DALLAS', '#003594', '#869397'], DEN: ['DENVER', '#FB4F14', '#002244'],
  DET: ['DETROIT', '#0076B6', '#B0B7BC'], GB: ['GREEN BAY', '#203731', '#FFB612'],
  HOU: ['HOUSTON', '#03202F', '#A71930'], IND: ['INDIANAPOLIS', '#002C5F', '#A2AAAD'],
  JAX: ['JACKSONVILLE', '#006778', '#D7A22A'], KC: ['KANSAS CITY', '#E31837', '#FFB81C'],
  LV: ['LAS VEGAS', '#000000', '#A5ACAF'], LAC: ['LOS ANGELES', '#0080C6', '#FFC20E'],
  LAR: ['LOS ANGELES', '#003594', '#FFA300'], MIA: ['MIAMI', '#008E97', '#FC4C02'],
  MIN: ['MINNESOTA', '#4F2683', '#FFC62F'], NE: ['NEW ENGLAND', '#002244', '#C60C30'],
  NO: ['NEW ORLEANS', '#D3BC8D', '#101820'], NYG: ['NEW YORK', '#0B2265', '#A71930'],
  NYJ: ['NEW YORK', '#125740', '#FFFFFF'], PHI: ['PHILADELPHIA', '#004C54', '#A5ACAF'],
  PIT: ['PITTSBURGH', '#101820', '#FFB612'], SF: ['SAN FRANCISCO', '#AA0000', '#B3995D'],
  SEA: ['SEATTLE', '#002244', '#69BE28'], TB: ['TAMPA BAY', '#D50A0A', '#FF7900'],
  TEN: ['TENNESSEE', '#0C2340', '#4B92DB'], WSH: ['WASHINGTON', '#5A1414', '#FFB612'],
};

/** club: the club's colours by ESPN abbreviation; an unknown club gets neutral greys and its own letters. */
export function club(ab: string): Club {
  const c = CLUBS[ab];
  return c ? { ab, name: c[0], pri: c[1], sec: c[2] } : { ab, name: ab, pri: '#5B6270', sec: '#C9CED6' };
}

const hex = (h: string): RGB => [1, 3, 5].map(i => parseInt(h.slice(i, i + 2), 16)) as RGB;
const rgb = (c: RGB) => `rgb(${c.map(Math.round).join(',')})`;
const lum = (c: RGB) => {
  const f = (v: number) => { v /= 255; return v <= .03928 ? v / 12.92 : Math.pow((v + .055) / 1.055, 2.4); };
  return .2126 * f(c[0]) + .7152 * f(c[1]) + .0722 * f(c[2]);
};
const contrast = (a: RGB, b: RGB) => {
  const x = lum(a), y = lum(b);
  return (Math.max(x, y) + .05) / (Math.min(x, y) + .05);
};
function toHsl([r, g, b]: RGB): RGB {
  r /= 255; g /= 255; b /= 255;
  const mx = Math.max(r, g, b), mn = Math.min(r, g, b), l = (mx + mn) / 2, d = mx - mn;
  if (!d) return [0, 0, l];
  const s = d / (1 - Math.abs(2 * l - 1));
  const h = mx === r ? ((g - b) / d + 6) % 6 : mx === g ? (b - r) / d + 2 : (r - g) / d + 4;
  return [h * 60, s, l];
}
function fromHsl(h: number, s: number, l: number): RGB {
  const c = (1 - Math.abs(2 * l - 1)) * s, x = c * (1 - Math.abs((h / 60) % 2 - 1)), m = l - c / 2;
  const i = Math.floor(h / 60) % 6;
  const parts: RGB[] = [[c, x, 0], [x, c, 0], [0, c, x], [0, x, c], [x, 0, c], [c, 0, x]];
  return parts[i].map(v => (v + m) * 255) as RGB;
}
const C_CANVAS: RGB = [12, 14, 18], C_BLACK: RGB = [4, 5, 6], C_WHITE: RGB = [240, 242, 245];
const TURF_RGB: RGB = [22, 47, 31];
/* ink: keep the club's hue and raise lightness only as far as the contrast floor needs. A club whose first
   colour is near-black is shown in its second colour, as it is on its own uniforms. */
function inkOn(pri: RGB, sec: RGB, bg = C_CANVAS, min = 5.5): RGB {
  let [h, s, l] = toHsl(pri);
  if (l < .14 || s < .15) { if (contrast(sec, bg) >= min) return sec; [h, s, l] = toHsl(sec); }
  s = Math.max(s, .5);
  let c = fromHsl(h, s, l);
  while (contrast(c, bg) < min && l < .96) { l += .01; c = fromHsl(h, s, l); }
  return c;
}
const textOn = (fill: RGB) => contrast(C_WHITE, fill) >= contrast(C_BLACK, fill) ? C_WHITE : C_BLACK;

export type ClubColours = { ab: string; name: string; color: string; fillc: string; onc: string;
  trimc: string; turf: string };
export function clubColours(t: Club): ClubColours {
  const pri = hex(t.pri), sec = hex(t.sec);
  /* a club colour painted on the turf: a green club on green turf uses its second colour */
  const green = toHsl(pri)[1] > .15 && Math.abs(toHsl(pri)[0] - 140) < 50;
  return { ab: t.ab, name: t.name, color: rgb(inkOn(pri, sec)), fillc: rgb(pri), onc: rgb(textOn(pri)),
    trimc: rgb(sec), turf: rgb(inkOn(green ? sec : pri, green ? pri : sec, TURF_RGB, 4.6)) };
}
