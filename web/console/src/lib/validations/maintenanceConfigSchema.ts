import * as yup from 'yup'

export const maintenanceConfigSchema = yup.object({
  enabled: yup.boolean().required(),
  message: yup.string().required('Message is required').max(500),
  scheduled_start: yup.string().nullable().default(null),
  scheduled_end: yup
    .string()
    .nullable()
    .default(null)
    // ValidateMaintenanceConfig in internal/tenant/validation_setting.go refuses
    // a window whose start is not strictly before its end. Checking it here puts
    // the message on the offending field instead of surfacing the rejection as a
    // bare error toast after the round trip.
    .test('after-scheduled-start', 'Scheduled end must be after scheduled start', function (value) {
      const start = this.parent.scheduled_start
      if (!start || !value) return true
      const startTime = new Date(start).getTime()
      const endTime = new Date(value).getTime()
      // An unparseable half of the range is not this rule's failure to report.
      if (Number.isNaN(startTime) || Number.isNaN(endTime)) return true
      return startTime < endTime
    }),
}).required()

export type MaintenanceConfigFormData = yup.InferType<typeof maintenanceConfigSchema>
