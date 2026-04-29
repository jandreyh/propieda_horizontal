export default function AnnouncementsPage() {
  const placeholderItems = [
    {
      id: "ANN-001",
      title: "Corte de agua programado",
      content: "Se realizara mantenimiento en el tanque principal el 3 de mayo de 8:00 a 14:00.",
      author: "Administracion",
      date: "2026-04-28",
      priority: "Alta",
    },
    {
      id: "ANN-002",
      title: "Horario de piscina actualizado",
      content: "A partir del 1 de mayo, la piscina estara disponible de 6:00 a 20:00.",
      author: "Administracion",
      date: "2026-04-25",
      priority: "Normal",
    },
    {
      id: "ANN-003",
      title: "Fumigacion de zonas comunes",
      content: "Se realizara fumigacion el sabado 4 de mayo. Se recomienda mantener mascotas dentro de los apartamentos.",
      author: "Administracion",
      date: "2026-04-22",
      priority: "Alta",
    },
    {
      id: "ANN-004",
      title: "Nuevo horario de porteria",
      content: "El cambio de turno de porteria sera a las 6:00, 14:00 y 22:00.",
      author: "Seguridad",
      date: "2026-04-20",
      priority: "Normal",
    },
  ];

  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">Anuncios</h1>
      <p className="mb-6 text-sm text-gray-500">
        Tablero de anuncios y comunicados de la comunidad
      </p>

      <div className="space-y-4">
        {placeholderItems.map((item) => (
          <div
            key={item.id}
            className="rounded-lg border border-gray-200 bg-white p-5"
          >
            <div className="mb-2 flex items-center justify-between">
              <h2 className="text-base font-semibold text-gray-900">
                {item.title}
              </h2>
              <span
                className={`inline-block rounded-full px-2 py-1 text-xs font-medium ${
                  item.priority === "Alta"
                    ? "bg-red-100 text-red-700"
                    : "bg-gray-100 text-gray-700"
                }`}
              >
                {item.priority}
              </span>
            </div>
            <p className="mb-3 text-sm text-gray-700">{item.content}</p>
            <div className="flex items-center gap-4 text-xs text-gray-500">
              <span>Por: {item.author}</span>
              <span>{item.date}</span>
              <span className="text-gray-400">{item.id}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
