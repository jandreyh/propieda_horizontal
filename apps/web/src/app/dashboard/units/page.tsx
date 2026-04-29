export default function UnitsPage() {
  const placeholderItems = [
    { id: "UNT-001", code: "Apto 301", block: "Torre A", floor: "3", area: "72 m2", type: "Apartamento", status: "Ocupado" },
    { id: "UNT-002", code: "Apto 502", block: "Torre A", floor: "5", area: "85 m2", type: "Apartamento", status: "Ocupado" },
    { id: "UNT-003", code: "Apto 805", block: "Torre B", floor: "8", area: "72 m2", type: "Apartamento", status: "Vacante" },
    { id: "UNT-004", code: "Local 2", block: "Comercial", floor: "1", area: "45 m2", type: "Local", status: "Ocupado" },
    { id: "UNT-005", code: "Apto 1102", block: "Torre B", floor: "11", area: "95 m2", type: "Apartamento", status: "Ocupado" },
    { id: "UNT-006", code: "Apto 402", block: "Torre A", floor: "4", area: "72 m2", type: "Apartamento", status: "Ocupado" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Unidades</h1>
      <p className="mb-6 text-sm text-gray-500">
        Apartamentos, casas, locales y demas unidades del conjunto
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Codigo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Bloque</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Piso</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Area</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Tipo</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.code}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.block}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.floor}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.area}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.type}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Ocupado"
                        ? "bg-green-100 text-green-700"
                        : "bg-yellow-100 text-yellow-700"
                    }`}
                  >
                    {item.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
