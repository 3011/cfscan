"use client"

import * as React from "react"
import { mergeProps } from "@base-ui/react/merge-props"
import {
  Controller,
  FormProvider,
  useFormContext,
  type ControllerProps,
  type FieldPath,
  type FieldValues,
} from "react-hook-form"

import { cn } from "@/lib/utils"
import { Label } from "@/components/ui/label"

const Form = FormProvider

type FormFieldContextValue<
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
> = { name: TName }

const FormFieldContext = React.createContext<FormFieldContextValue | null>(null)

const FormField = <
  TFieldValues extends FieldValues = FieldValues,
  TName extends FieldPath<TFieldValues> = FieldPath<TFieldValues>,
>(props: ControllerProps<TFieldValues, TName>) => (
  <FormFieldContext.Provider value={{ name: props.name }}>
    <Controller {...props} />
  </FormFieldContext.Provider>
)

type FormItemContextValue = { id: string }
const FormItemContext = React.createContext<FormItemContextValue | null>(null)

function useFormField() {
  const fieldContext = React.useContext(FormFieldContext)
  const itemContext = React.useContext(FormItemContext)
  const { getFieldState, formState } = useFormContext()
  if (!fieldContext) throw new Error("useFormField should be used within <FormField>")
  if (!itemContext) throw new Error("useFormField should be used within <FormItem>")
  const fieldState = getFieldState(fieldContext.name, formState)
  const { id } = itemContext
  return {
    id,
    name: fieldContext.name,
    formItemId: `${id}-form-item`,
    formDescriptionId: `${id}-form-item-description`,
    formMessageId: `${id}-form-item-message`,
    ...fieldState,
  }
}

function FormItem({ className, ...props }: React.ComponentProps<"div">) {
  const id = React.useId()
  return (
    <FormItemContext.Provider value={{ id }}>
      <div data-slot="form-item" className={cn("grid gap-2", className)} {...props} />
    </FormItemContext.Provider>
  )
}

function FormLabel({ className, ...props }: React.ComponentProps<typeof Label>) {
  const { error, formItemId } = useFormField()
  return <Label data-slot="form-label" className={cn(error && "text-destructive", className)} htmlFor={formItemId} {...props} />
}

type FormControlProps = React.HTMLAttributes<HTMLElement> & { children: React.ReactElement }

const FormControl = React.forwardRef<HTMLElement, FormControlProps>(
  ({ children, ...props }, forwardedRef) => {
    const { error, formItemId, formDescriptionId, formMessageId } = useFormField()
    const child = React.Children.only(children) as React.ReactElement<Record<string, unknown> & { ref?: React.Ref<HTMLElement> }>
    const childRef = child.props.ref
    const mergedRef = React.useCallback((node: HTMLElement | null) => {
      for (const ref of [childRef, forwardedRef]) {
        if (typeof ref === "function") ref(node)
        else if (ref) ref.current = node
      }
    }, [childRef, forwardedRef])
    const merged = mergeProps(
      child.props,
      props,
      {
        id: formItemId,
        "aria-describedby": !error ? formDescriptionId : `${formDescriptionId} ${formMessageId}`,
        "aria-invalid": Boolean(error),
        "data-slot": "form-control",
      },
    )
    return React.cloneElement(child, { ...merged, ref: mergedRef })
  },
)
FormControl.displayName = "FormControl"

function FormDescription({ className, ...props }: React.ComponentProps<"p">) {
  const { formDescriptionId } = useFormField()
  return <p data-slot="form-description" id={formDescriptionId} className={cn("text-xs text-muted-foreground", className)} {...props} />
}

function FormMessage({ className, children, ...props }: React.ComponentProps<"p">) {
  const { error, formMessageId } = useFormField()
  const body = error ? String(error.message ?? "") : children
  if (!body) return null
  return <p data-slot="form-message" id={formMessageId} className={cn("text-xs font-medium text-destructive", className)} {...props}>{body}</p>
}

export { useFormField, Form, FormItem, FormLabel, FormControl, FormDescription, FormMessage, FormField }
