Imports System.IO

Namespace Vanni.Idmix

''' <summary>XID v1.1 二进制层编解码。</summary>
Public NotInheritable Class XidCodec
    Private Shared ReadOnly SwBytes As Integer() = {1, 2, 4, 8}
    Private Shared ReadOnly EmbeddedOType As Integer(,) = {
        {TypedValue.OTypeUint8, TypedValue.OTypeUint16, TypedValue.OTypeUint32, TypedValue.OTypeUint64},
        {TypedValue.OTypeInt8, TypedValue.OTypeInt16, TypedValue.OTypeInt32, TypedValue.OTypeInt64}
    }

    Private Sub New()
    End Sub

    Public Shared Function EncodeBinary(m As IdMix, typed As IList(Of TypedValue), variantId As Integer) As Byte()
        Using objects As New MemoryStream(typed.Count * 9)
            For Each tv In typed
            Dim obj = EncodeObject(tv)
            objects.Write(obj, 0, obj.Length)
        Next
            Dim objBytes = objects.ToArray()
            Dim mask = CByte((variantId * &H9D + &H37) And &HFF)
            For i = 0 To objBytes.Length - 1
                objBytes(i) = objBytes(i) Xor mask
            Next
            Dim count = typed.Count
            Dim header = (variantId << m.VariantShift) Or (count << m.CountShift)
            Dim data(1 + objBytes.Length) As Byte
            data(0) = CByte(header And &HFF)
            data(1) = CByte((header >> 8) And &HFF)
            Buffer.BlockCopy(objBytes, 0, data, 2, objBytes.Length)
            Dim xorSum = 0
            For Each b In data
                xorSum = xorSum Xor b
            Next
            header = header Or (xorSum And m.CheckMask)
            data(0) = CByte(header And &HFF)
            data(1) = CByte((header >> 8) And &HFF)
            Return data
        End Using
    End Function

    Public Shared Function DecodeBinary(m As IdMix, data As Byte()) As List(Of TypedValue)
        If data.Length < 2 Then Throw New ArgumentException("invalid data: too short")
        Dim header = data(0) Or (data(1) << 8)
        Dim check = header And m.CheckMask
        Dim count = (header And m.CountMask) >> m.CountShift
        Dim variantId = (header And m.VariantMask) >> m.VariantShift
        If variantId >= m.MaxVariants Then Throw New ArgumentException($"invalid variant_id {variantId}")
        If count > m.MaxObjects Then Throw New ArgumentException($"invalid count {count}")
        Dim verify = CType(data.Clone(), Byte())
        verify(0) = CByte(verify(0) And Not m.CheckMask)
        Dim xorSum = 0
        For Each b In verify
            xorSum = xorSum Xor b
        Next
        If (xorSum And m.CheckMask) <> check Then Throw New ArgumentException("checksum mismatch")
        Dim objects(data.Length - 3) As Byte
        Buffer.BlockCopy(data, 2, objects, 0, objects.Length)
        Dim mask = CByte((variantId * &H9D + &H37) And &HFF)
        For i = 0 To objects.Length - 1
            objects(i) = objects(i) Xor mask
        Next
        Dim result As New List(Of TypedValue)()
        Dim pos = 0
        For i = 0 To count - 1
            If pos >= objects.Length Then Throw New ArgumentException("premature end of data")
            Dim dr = DecodeObject(objects, pos)
            result.Add(dr.Tv)
            pos += dr.Consumed
        Next
        If pos <> objects.Length Then Throw New ArgumentException("extra bytes after data objects")
        Return result
    End Function

    Private Shared Function EncodeObject(tv As TypedValue) As Byte()
        ValidateRange(tv.OType, tv.Val)
        If IsUnsigned(tv.OType) AndAlso tv.Val >= 0 AndAlso tv.Val <= 15 Then
            Dim wb = WidthBits(tv.OType)
            Return {CByte((wb << 4) Or tv.Val)}
        End If
        If IsSigned(tv.OType) AndAlso tv.Val >= -16 AndAlso tv.Val <= -1 Then
            Dim wb = WidthBits(tv.OType)
            Dim v = CInt(-tv.Val - 1)
            Return {CByte((1 << 6) Or (wb << 4) Or v)}
        End If
        Dim swPayload = MinimalComplementBytes(tv.OType, tv.Val)
        Dim sw = swPayload(0)
        Dim outBuf(swPayload.Length - 1) As Byte
        outBuf(0) = CByte(&H80 Or (sw << 4) Or tv.OType)
        Buffer.BlockCopy(swPayload, 1, outBuf, 1, swPayload.Length - 1)
        Return outBuf
    End Function

    Private Structure DecodeResult
        Public Tv As TypedValue
        Public Consumed As Integer
    End Structure

    Private Shared Function DecodeObject(data As Byte(), offset As Integer) As DecodeResult
        If offset >= data.Length Then Throw New ArgumentException("truncated object header")
        Dim head = data(offset)
        If (head And &H80) = 0 Then
            Dim sign = (head >> 6) And 1
            Dim wb = (head >> 4) And &H03
            Dim v = head And &H0F
            Dim otype = EmbeddedOType(sign, wb)
            Dim val = If(sign = 0, CLng(v), -v - 1L)
            Return New DecodeResult With {.Tv = New TypedValue(otype, val), .Consumed = 1}
        End If
        If ((head >> 6) And 1) <> 0 Then Throw New ArgumentException("reserved bit set in extended mode")
        Dim sw = (head >> 4) And &H03
        Dim otype2 = head And &H0F
        If otype2 > TypedValue.OTypeInt64 Then Throw New ArgumentException($"invalid otype {otype2}")
        Dim numBytes = SwBytes(sw)
        If data.Length < offset + 1 + numBytes Then Throw New ArgumentException("truncated object payload")
        Dim raw As Long = 0
        For i = 0 To numBytes - 1
            raw = raw Or (CLng(data(offset + 1 + i) And &HFF) << (8 * i))
        Next
        Dim val2 = ReconstructInt(otype2, sw, raw)
        Return New DecodeResult With {.Tv = New TypedValue(otype2, val2), .Consumed = 1 + numBytes}
    End Function

    Private Shared Function IsUnsigned(otype As Integer) As Boolean
        Return otype <= TypedValue.OTypeUint64
    End Function

    Private Shared Function IsSigned(otype As Integer) As Boolean
        Return otype >= TypedValue.OTypeInt8
    End Function

    Private Shared Function WidthBits(otype As Integer) As Integer
        Select Case otype
            Case TypedValue.OTypeUint8, TypedValue.OTypeInt8 : Return 0
            Case TypedValue.OTypeUint16, TypedValue.OTypeInt16 : Return 1
            Case TypedValue.OTypeUint32, TypedValue.OTypeInt32 : Return 2
            Case Else : Return 3
        End Select
    End Function

    Private Shared Function TargetBits(otype As Integer) As Integer
        Select Case otype
            Case TypedValue.OTypeUint8, TypedValue.OTypeInt8 : Return 8
            Case TypedValue.OTypeUint16, TypedValue.OTypeInt16 : Return 16
            Case TypedValue.OTypeUint32, TypedValue.OTypeInt32 : Return 32
            Case Else : Return 64
        End Select
    End Function

    Private Shared Function MinimalComplementBytes(otype As Integer, val As Long) As Integer()
        If val = 0 Then Return {0, 0}
        If IsUnsigned(otype) Then
            If val < 0 Then Throw New ArgumentException("negative value for unsigned type")
            For sw = 0 To 3
                Dim size = SwBytes(sw)
                If size < 8 AndAlso val >= (1L << (size * 8)) Then Continue For
                Dim buf = UintToLeBytes(val, size)
                If (buf(size - 1) And &H80) = 0 Then
                    Dim outArr(1 + size - 1) As Integer
                    outArr(0) = sw
                    Buffer.BlockCopy(buf, 0, outArr, 1, size)
                    Return outArr
                End If
            Next
            Throw New ArgumentException("value too large for unsigned type")
        End If
        Dim tbits = TargetBits(otype)
        Dim mask = If(tbits = 64, -1L, (1L << tbits) - 1)
        Dim uval = val And mask
        If val < 0 Then
            For sw = 0 To 3
                Dim size = SwBytes(sw)
                Dim shift = size * 8
                If shift >= tbits Then
                    Dim buf = UintToLeBytes(uval, size)
                    Dim outArr(1 + size - 1) As Integer
                    outArr(0) = sw
                    Buffer.BlockCopy(buf, 0, outArr, 1, size)
                    Return outArr
                End If
                Dim lower = uval And ((1L << shift) - 1)
                Dim upper = uval >> shift
                Dim upperMask = (1L << (tbits - shift)) - 1
                If upper <> upperMask Then Continue For
                Dim highByte = CInt((lower >> (shift - 8)) And &HFF)
                If (highByte And &H80) = 0 Then Continue For
                Dim buf2 = UintToLeBytes(lower, size)
                Dim outArr2(1 + size - 1) As Integer
                outArr2(0) = sw
                Buffer.BlockCopy(buf2, 0, outArr2, 1, size)
                Return outArr2
            Next
        Else
            For sw = 0 To 3
                Dim size = SwBytes(sw)
                If size < 8 AndAlso uval >= (1L << (size * 8)) Then Continue For
                Dim buf = UintToLeBytes(uval, size)
                If (buf(size - 1) And &H80) = 0 Then
                    Dim outArr(1 + size - 1) As Integer
                    outArr(0) = sw
                    Buffer.BlockCopy(buf, 0, outArr, 1, size)
                    Return outArr
                End If
            Next
        End If
        Dim swFinal = If(tbits = 8, 0, If(tbits = 16, 1, If(tbits = 32, 2, 3)))
        Dim bufFinal = UintToLeBytes(uval, SwBytes(swFinal))
        Dim outFinal(1 + bufFinal.Length - 1) As Integer
        outFinal(0) = swFinal
        Buffer.BlockCopy(bufFinal, 0, outFinal, 1, bufFinal.Length)
        Return outFinal
    End Function

    Private Shared Function UintToLeBytes(v As Long, size As Integer) As Byte()
        Dim buf(size - 1) As Byte
        For i = 0 To size - 1
            buf(i) = CByte((v >> (8 * i)) And &HFF)
        Next
        Return buf
    End Function

    Private Shared Function ReconstructInt(otype As Integer, sw As Integer, raw As Long) As Long
        Dim tbits = TargetBits(otype)
        Dim storedBits = SwBytes(sw) * 8
        If IsUnsigned(otype) Then
            Dim mask = If(tbits = 64, -1L, (1L << tbits) - 1)
            Return raw And mask
        End If
        Dim signBit = CInt((raw >> (storedBits - 1)) And 1)
        If tbits <= storedBits Then
            Dim mask = (1L << tbits) - 1
            Dim val = raw And mask
            If signBit = 1 AndAlso (val And (1L << (tbits - 1))) <> 0 Then val -= 1L << tbits
            Return val
        End If
        Dim extended As Long
        If signBit = 1 Then
            Dim extendMask = (Not ((1L << storedBits) - 1)) And ((1L << tbits) - 1)
            extended = raw Or extendMask
        Else
            extended = raw
        End If
        If extended >= (1L << (tbits - 1)) Then extended -= 1L << tbits
        Return extended
    End Function

    Private Shared Sub ValidateRange(otype As Integer, val As Long)
        Dim ok As Boolean
        Select Case otype
            Case TypedValue.OTypeUint8 : ok = val >= 0 AndAlso val <= &HFF
            Case TypedValue.OTypeUint16 : ok = val >= 0 AndAlso val <= &HFFFF
            Case TypedValue.OTypeUint32 : ok = val >= 0 AndAlso val <= &HFFFFFFFFL
            Case TypedValue.OTypeUint64 : ok = val >= 0
            Case TypedValue.OTypeInt8 : ok = val >= -128 AndAlso val <= 127
            Case TypedValue.OTypeInt16 : ok = val >= -32768 AndAlso val <= 32767
            Case TypedValue.OTypeInt32 : ok = val >= Integer.MinValue AndAlso val <= Integer.MaxValue
            Case TypedValue.OTypeInt64 : ok = True
            Case Else : ok = False
        End Select
        If Not ok Then Throw New ArgumentException($"value {val} out of range for otype {otype}")
    End Sub
End Class

End Namespace
