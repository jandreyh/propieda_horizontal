export default function FinancePage() {
  const placeholderItems = [
    { id: "FIN-001", concept: "Cuota administracion - Abril 2026", amount: "$350,000", status: "Pendiente" },
    { id: "FIN-002", concept: "Cuota extraordinaria - Ascensor", amount: "$120,000", status: "Pagado" },
    { id: "FIN-003", concept: "Cuota administracion - Marzo 2026", amount: "$350,000", status: "Pagado" },
    { id: "FIN-004", concept: "Multa estacionamiento", amount: "$50,000", status: "Vencido" },
    { id: "FIN-005", concept: "Cuota administracion - Febrero 2026", amount: "$340,000", status: "Pagado" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Finanzas</h1>
      <p className="mb-6 text-sm text-gray-500">
        Facturacion, pagos y cartera de cobros del conjunto
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Concepto</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Monto</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.concept}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.amount}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Pagado"
                        ? "bg-green-100 text-green-700"
                        : item.status === "Pendiente"
                          ? "bg-yellow-100 text-yellow-700"
                          : "bg-red-100 text-red-700"
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
