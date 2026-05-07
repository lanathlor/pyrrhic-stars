import { useQuery } from "@tanstack/react-query";
import { fetchInstance, fetchEvents } from "../api";

export function useFightData(instanceId: string) {
  const instanceQuery = useQuery({
    queryKey: ["instance", instanceId],
    queryFn: () => fetchInstance(instanceId),
  });

  const eventsQuery = useQuery({
    queryKey: ["events", instanceId],
    queryFn: () => fetchEvents(instanceId),
  });

  return {
    instance: instanceQuery.data ?? null,
    events: eventsQuery.data ?? [],
    loading: instanceQuery.isLoading || eventsQuery.isLoading,
    error: instanceQuery.error?.message ?? eventsQuery.error?.message ?? "",
  };
}
