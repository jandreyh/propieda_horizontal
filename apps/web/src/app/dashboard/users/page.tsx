export default function UsersPage() {
  const placeholderItems = [
    { id: "USR-001", name: "Carlos Rodriguez", document: "CC 1.234.567", unit: "Apto 301", role: "Propietario", status: "Activo" },
    { id: "USR-002", name: "Ana Martinez", document: "CC 9.876.543", unit: "Apto 502", role: "Arrendatario", status: "Activo" },
    { id: "USR-003", name: "Luis Gomez", document: "CC 5.555.555", unit: "--", role: "Administrador", status: "Activo" },
    { id: "USR-004", name: "Maria Fernandez", document: "CC 3.333.333", unit: "Apto 805", role: "Propietario", status: "Inactivo" },
    { id: "USR-005", name: "Jorge Diaz", document: "CC 7.777.777", unit: "--", role: "Portero", status: "Activo" },
    { id: "USR-006", name: "Sandra Lopez", document: "CC 2.222.222", unit: "Local 2", role: "Propietario", status: "Activo" },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Usuarios</h1>
      <p className="mb-6 text-sm text-gray-500">
        Residentes, administradores y personal del conjunto
      </p>

      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">ID</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Nombre</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Documento</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Unidad</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Rol</th>
              <th className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">Estado</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {placeholderItems.map((item) => (
              <tr key={item.id} className="hover:bg-gray-50">
                <td className="whitespace-nowrap px-6 py-4 text-sm font-medium text-gray-900">{item.id}</td>
                <td className="px-6 py-4 text-sm text-gray-700">{item.name}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.document}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.unit}</td>
                <td className="whitespace-nowrap px-6 py-4 text-sm text-gray-700">{item.role}</td>
                <td className="whitespace-nowrap px-6 py-4">
                  <span
                    className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                      item.status === "Activo"
                        ? "bg-green-100 text-green-700"
                        : "bg-gray-100 text-gray-700"
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
