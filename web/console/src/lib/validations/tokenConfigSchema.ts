import * as yup from 'yup'

export const tokenConfigSchema = yup.object({
  clock_skew_leeway_seconds: yup
    .number()
    .required()
    .min(0, 'Cannot be negative')
    .max(300, 'Cannot exceed 300'),
  signing_algorithm: yup
    .string()
    .required()
    // ES256 is intentionally omitted: the server signs with an RSA key store and
    // rejects ES256, so offering it here only lets an operator pick a value that
    // the API refuses (and would break token issuance if it slipped through).
    .oneOf(['RS256', 'PS256']),
  require_pkce: yup.boolean().required(),
  additional_id_token_claims: yup.array().of(yup.string().defined()).required(),
  additional_access_token_claims: yup.array().of(yup.string().defined()).required(),
}).required()

export type TokenConfigFormData = yup.InferType<typeof tokenConfigSchema>
