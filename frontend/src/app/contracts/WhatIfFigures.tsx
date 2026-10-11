import type { WhatIf } from './whatif';

export function WhatIfFigures({ figures, label }: { figures: WhatIf['figures']; label: string }) {
  return <table className="whatif-figures" aria-label={label}>
    <thead><tr><th>Result</th><th>Before</th><th>After</th><th>Change</th></tr></thead>
    <tbody>{figures.map(figure => <tr key={figure.label}>
      <th scope="row">{figure.label}</th>
      <td className="contract-money">{figure.before}</td>
      <td className="contract-money">{figure.after}</td>
      <td className="contract-money">{figure.change}</td>
    </tr>)}</tbody>
  </table>;
}
