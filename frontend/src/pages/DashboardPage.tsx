import Card from "../components/ui/Card";
import Badge from "../components/ui/Badge";

// Temporary dashboard placeholder. Real widgets arrive in M5.
export default function DashboardPage() {
  return (
    <div className="space-y-6">
      <h1 className="text-title-md font-semibold text-gray-800 dark:text-white/90">
        Dashboard
      </h1>

      <Card title="UI kit check" desc="TailAdmin design system ported to Vite.">
        <div className="flex flex-wrap items-center gap-2">
          <Badge color="primary">primary</Badge>
          <Badge color="success">success</Badge>
          <Badge color="error">error</Badge>
          <Badge color="warning">warning</Badge>
          <Badge variant="solid" color="primary">solid</Badge>
        </div>
      </Card>
    </div>
  );
}
