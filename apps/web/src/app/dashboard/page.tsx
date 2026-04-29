import Link from "next/link";

interface ModuleCard {
  title: string;
  description: string;
  href: string;
}

const modules: ModuleCard[] = [
  {
    title: "Finanzas",
    description: "Facturacion, pagos y cartera de cobros",
    href: "/dashboard/finance",
  },
  {
    title: "Reservas",
    description: "Reserva de areas comunes y zonas sociales",
    href: "/dashboard/reservations",
  },
  {
    title: "Asambleas",
    description: "Reuniones de copropietarios y votaciones",
    href: "/dashboard/assemblies",
  },
  {
    title: "Incidentes",
    description: "Reporte y seguimiento de incidentes",
    href: "/dashboard/incidents",
  },
  {
    title: "Sanciones",
    description: "Multas y sanciones a infractores",
    href: "/dashboard/penalties",
  },
  {
    title: "PQRS",
    description: "Peticiones, quejas, reclamos y sugerencias",
    href: "/dashboard/pqrs",
  },
  {
    title: "Notificaciones",
    description: "Alertas y preferencias de comunicacion",
    href: "/dashboard/notifications",
  },
  {
    title: "Parqueaderos",
    description: "Gestion de espacios de parqueo",
    href: "/dashboard/parking",
  },
  {
    title: "Paquetes",
    description: "Correspondencia y paqueteria",
    href: "/dashboard/packages",
  },
  {
    title: "Control de acceso",
    description: "Operaciones de porteria y visitantes",
    href: "/dashboard/access-control",
  },
  {
    title: "Anuncios",
    description: "Tablero de anuncios de la comunidad",
    href: "/dashboard/announcements",
  },
  {
    title: "Unidades",
    description: "Apartamentos, casas y locales",
    href: "/dashboard/units",
  },
  {
    title: "Usuarios",
    description: "Residentes, administradores y personal",
    href: "/dashboard/users",
  },
];

export default function DashboardPage() {
  return (
    <div>
      <h1 className="mb-1 text-2xl font-bold text-gray-900">
        Panel de administracion
      </h1>
      <p className="mb-6 text-sm text-gray-500">
        Seleccione un modulo para comenzar
      </p>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {modules.map((mod) => (
          <Link
            key={mod.href}
            href={mod.href}
            className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md"
          >
            <h2 className="mb-1 text-base font-semibold text-gray-900">
              {mod.title}
            </h2>
            <p className="text-sm text-gray-500">{mod.description}</p>
          </Link>
        ))}
      </div>
    </div>
  );
}
